package consumers

import (
	"errors"

	"gopkg.in/alecthomas/kingpin.v2"

	"strings"

	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

// Implementations provide access to AWS Route53 API's
// required calls.
type AWSClient interface {
	ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error
	EndpointsToAlias(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error)
	RecordInfo(records []*route53.ResourceRecordSet) map[string]*pkg.RecordInfo
	GetGroupID() string
	GetAssignedTXTRecordObject(record *route53.ResourceRecordSet) *route53.ResourceRecordSet
	GetHostedZones() (map[string]string, error)
}

type awsClient struct {
	client AWSClient
}

func init() {
	kingpin.Flag("aws-record-group-id", "Identifier to filter the mate records ").StringVar(&params.awsGroupID)
}

// NewAWSRoute53 reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSRoute53() (Consumer, error) {
	if params.awsGroupID == "" {
		return nil, errors.New("please provide --aws-record-group-id")
	}
	return withClient(awsclient.New(awsclient.Options{
		GroupID: params.awsGroupID,
	})), nil
}

func withClient(c AWSClient) *awsClient {
	return &awsClient{c}
}

func (a *awsClient) Sync(endpoints []*pkg.Endpoint) error {
	newAliasRecords, err := a.client.EndpointsToAlias(endpoints)
	if err != nil {
		return err
	}

	hostedZonesMap, err := a.client.GetHostedZones()
	if err != nil {
		return err
	}
	if len(hostedZonesMap) == 0 {
		log.Warnln("No hosted zones found in Route53. At least one hosted zone should be created to create DNS records...")
		return nil
	}

	inputByZoneID := map[string][]*route53.ResourceRecordSet{}
	for _, record := range newAliasRecords {
		zoneID := getZoneIDForEndpoint(hostedZonesMap, record) //this guarantees that the endpoint will not be created in multiple hosted zones
		if zoneID == "" {
			log.Warnf("Hosted zone for endpoint: %s is not found. Skipping record...", aws.StringValue(record.Name))
			continue
		}
		inputByZoneID[zoneID] = append(inputByZoneID[zoneID], record)
	}

	var wg sync.WaitGroup
	for zoneName, zoneID := range hostedZonesMap {
		if len(inputByZoneID[zoneID]) > 0 {
			wg.Add(1)
			zoneID := zoneID
			go func() {
				defer wg.Done()
				err := a.syncPerHostedZone(inputByZoneID[zoneID], zoneID)
				if err != nil {
					//should pass the err down the error channel
					//for now just log
					log.Errorf("Error changing records per zone: %s", zoneName)
				}
			}()
		}
	}
	wg.Wait()
	return nil
}

func (a *awsClient) syncPerHostedZone(newAliasRecords []*route53.ResourceRecordSet, zoneID string) error {
	existingRecords, err := a.client.ListRecordSets(zoneID)
	if err != nil {
		return err
	}

	recordInfoMap := a.client.RecordInfo(existingRecords)

	var upsert, del []*route53.ResourceRecordSet

	//find records to be upserted
	for _, newAliasRecord := range newAliasRecords {
		existingRecordInfo, exist := recordInfoMap[aws.StringValue(newAliasRecord.Name)]

		if !exist { //record does not exist, create it
			newTXTRecord := a.client.GetAssignedTXTRecordObject(newAliasRecord)
			upsert = append(upsert, newAliasRecord, newTXTRecord)
			continue
		}

		if existingRecordInfo.GroupID != a.client.GetGroupID() { // there exist a record with a different or empty group ID
			log.Warnf("Skipping record %s: with a group ID: %s", aws.StringValue(newAliasRecord.Name), existingRecordInfo.GroupID)
			continue
		}

		// make sure record only updated when target changes, not to spam AWS route53 API with dummy updates
		if pkg.SanitizeDNSName(existingRecordInfo.Target) != aws.StringValue(newAliasRecord.AliasTarget.DNSName) {
			newTXTRecord := a.client.GetAssignedTXTRecordObject(newAliasRecord)
			upsert = append(upsert, newAliasRecord, newTXTRecord)
		}
	}

	//find records to be removed
	for _, existingRecord := range existingRecords {
		recordInfo := recordInfoMap[aws.StringValue(existingRecord.Name)]
		if recordInfo.GroupID == a.client.GetGroupID() {
			remove := true
			for _, newAliasRecord := range newAliasRecords {
				if aws.StringValue(newAliasRecord.Name) == aws.StringValue(existingRecord.Name) {
					remove = false
				}
			}
			if remove {
				del = append(del, existingRecord)
			}
		}
	}

	if len(upsert) > 0 || len(del) > 0 {
		log.Debugln("Records to be upserted: ", upsert)
		log.Debugln("Records to be deleted: ", del)
		return a.client.ChangeRecordSets(upsert, del, nil, zoneID)
	}

	log.Infoln("No changes submitted")
	return nil
}

func (a *awsClient) Consume(endpoints chan *pkg.Endpoint, errors chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	log.Infoln("[AWS] Listening for events...")

	for {
		select {
		case e, ok := <-endpoints:
			if !ok {
				log.Info("[AWS] channel closed")
				return
			}

			log.Infof("[AWS] Processing (%s, %s, %s)\n", e.DNSName, e.IP, e.Hostname)

			err := a.Process(e)
			if err != nil {
				errors <- err
			}
		case <-done:
			log.Info("[AWS] Exited consuming loop.")
			return
		}
	}
}

func (a *awsClient) Process(endpoint *pkg.Endpoint) error {
	hostedZonesMap, err := a.client.GetHostedZones()
	if err != nil {
		return err
	}

	aliasRecords, err := a.client.EndpointsToAlias([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}

	create := []*route53.ResourceRecordSet{aliasRecords[0], a.client.GetAssignedTXTRecordObject(aliasRecords[0])}

	zoneID := getZoneIDForEndpoint(hostedZonesMap, aliasRecords[0])
	if zoneID == "" {
		log.Warnf("Hosted zone for endpoint: %s is not found. Skipping record...", endpoint.DNSName)
		return nil
	}

	err = a.client.ChangeRecordSets(nil, nil, create, zoneID)
	if err != nil && strings.Contains(err.Error(), "already exists") {
		log.Warnf("Record [name=%s] could not be created, another record with same name already exists", endpoint.DNSName)
		return nil
	}

	return err
}

//getZoneIDForEndpoint returns the zone id for the record based on its dns name, returns best match
//i.e. if the record has dns name "test.sub.example.com" and route53 has two hosted zones "example.com" and "sub.example.com"
//"sub.example.com" will be returned
func getZoneIDForEndpoint(hostedZonesMap map[string]string, record *route53.ResourceRecordSet) string {
	var match string
	for zoneName, zoneID := range hostedZonesMap {
		if strings.HasSuffix(aws.StringValue(record.Name), zoneName) && len(zoneName) > len(match) { //get the longest match for the dns name
			match = zoneID
		}
	}
	return match
}
