package consumers

import (
	"errors"
	"fmt"

	"gopkg.in/alecthomas/kingpin.v2"

	"strings"

	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

type AWSClient interface {
	ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error
	GetCanonicalZoneIDs(lbDNS []string) (map[string]string, error)
	GetHostedZones() (map[string]string, error)
}

type awsConsumer struct {
	groupID string
	client  AWSClient
}

const (
	evaluateTargetHealth = true
	defaultTxtTTL        = int64(300)
)

func init() {
	kingpin.Flag("aws-record-group-id", "Identifier to filter the mate records ").StringVar(&params.awsGroupID)
}

// NewAWSConsumer reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSConsumer() (Consumer, error) {
	if params.awsGroupID == "" {
		return nil, errors.New("please provide --aws-record-group-id")
	}
	return withClient(awsclient.New(awsclient.Options{}), params.awsGroupID), nil
}

func withClient(c AWSClient, groupID string) *awsConsumer {
	return &awsConsumer{
		groupID: groupID,
		client:  c,
	}
}

func (a *awsConsumer) Sync(endpoints []*pkg.Endpoint) error {
	newAliasRecords, err := a.endpointsToAlias(endpoints)
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
		wg.Add(1)
		go func(zoneName, zoneID string) {
			defer wg.Done()
			err := a.syncPerHostedZone(inputByZoneID[zoneID], zoneID)
			if err != nil {
				//should pass the err down the error channel
				//for now just log
				log.Errorf("Error changing records per zone: %s", zoneName)
			}
		}(zoneName, zoneID)
	}
	wg.Wait()
	return nil
}

func (a *awsConsumer) syncPerHostedZone(newAliasRecords []*route53.ResourceRecordSet, zoneID string) error {
	existingRecords, err := a.client.ListRecordSets(zoneID)
	if err != nil {
		return err
	}

	recordInfoMap := a.recordInfo(existingRecords)

	var upsert, del []*route53.ResourceRecordSet

	//find records to be upserted
	for _, newAliasRecord := range newAliasRecords {
		existingRecordInfo, exist := recordInfoMap[aws.StringValue(newAliasRecord.Name)]

		if !exist { //record does not exist, create it
			newTXTRecord := a.getAssignedTXTRecordObject(newAliasRecord)
			upsert = append(upsert, newAliasRecord, newTXTRecord)
			continue
		}

		if existingRecordInfo.GroupID != a.getGroupID() { // there exist a record with a different or empty group ID
			log.Warnf("Skipping record %s: with a group ID: %s", aws.StringValue(newAliasRecord.Name), existingRecordInfo.GroupID)
			continue
		}

		// make sure record only updated when target changes, not to spam AWS route53 API with dummy updates
		if pkg.SanitizeDNSName(existingRecordInfo.Target) != aws.StringValue(newAliasRecord.AliasTarget.DNSName) {
			newTXTRecord := a.getAssignedTXTRecordObject(newAliasRecord)
			upsert = append(upsert, newAliasRecord, newTXTRecord)
		}
	}

	//find records to be removed
	for _, existingRecord := range existingRecords {
		recordInfo := recordInfoMap[aws.StringValue(existingRecord.Name)]
		if recordInfo.GroupID == a.getGroupID() {
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

	log.Infoln("No changes submitted for zone: ", zoneID)
	return nil
}

func (a *awsConsumer) Consume(endpoints <-chan *pkg.Endpoint, errors chan<- error, done <-chan struct{}, wg *sync.WaitGroup) {
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

func (a *awsConsumer) Process(endpoint *pkg.Endpoint) error {
	hostedZonesMap, err := a.client.GetHostedZones()
	if err != nil {
		return err
	}

	aliasRecords, err := a.endpointsToAlias([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}

	create := []*route53.ResourceRecordSet{aliasRecords[0], a.getAssignedTXTRecordObject(aliasRecords[0])}

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
	var matchName string
	var matchID string
	for zoneName, zoneID := range hostedZonesMap {
		if strings.HasSuffix(aws.StringValue(record.Name), zoneName) && len(zoneName) > len(matchName) { //get the longest match for the dns name
			matchName = zoneName
			matchID = zoneID
		}
	}
	return matchID
}

//getGroupID returns the idenitifier for AWS records as stored in TXT records
func (a *awsConsumer) getGroupID() string {
	return fmt.Sprintf("\"mate:%s\"", a.groupID)
}

//getAssignedTXTRecordObject returns the TXT record which accompanies the Alias record
func (a *awsConsumer) getAssignedTXTRecordObject(aliasRecord *route53.ResourceRecordSet) *route53.ResourceRecordSet {
	return &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aliasRecord.Name,
		TTL:  aws.Int64(defaultTxtTTL),
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(a.getGroupID()),
		}},
	}
}

//recordInfo returns the map of record assigned dns to its target and groupID (can be empty)
func (a *awsConsumer) recordInfo(records []*route53.ResourceRecordSet) map[string]*pkg.RecordInfo {
	groupIDMap := map[string]string{} //maps dns to group ID

	for _, record := range records {
		if (aws.StringValue(record.Type)) == "TXT" {
			groupIDMap[aws.StringValue(record.Name)] = aws.StringValue(record.ResourceRecords[0].Value)
		} else {
			if _, exist := groupIDMap[aws.StringValue(record.Name)]; !exist {
				groupIDMap[aws.StringValue(record.Name)] = ""
			}
		}
	}

	infoMap := map[string]*pkg.RecordInfo{} //maps record DNS to its GroupID (if exists) and Target (LB)
	for _, record := range records {
		groupID := groupIDMap[aws.StringValue(record.Name)]
		if _, exist := infoMap[aws.StringValue(record.Name)]; !exist {
			infoMap[aws.StringValue(record.Name)] = &pkg.RecordInfo{
				GroupID: groupID,
			}
		}
		if aws.StringValue(record.Type) != "TXT" {
			infoMap[aws.StringValue(record.Name)].Target = a.getRecordTarget(record)
		}
	}

	return infoMap
}

//getRecordTarget returns the ELB dns for the given record
func (a *awsConsumer) getRecordTarget(r *route53.ResourceRecordSet) string {
	if aws.StringValue(r.Type) == "TXT" {
		return ""
	}
	if r.AliasTarget != nil {
		return aws.StringValue(r.AliasTarget.DNSName)
	}
	return aws.StringValue(r.ResourceRecords[0].Value)
}

//endpointsToAlias converts pkg Endpoint to route53 Alias Records
func (a *awsConsumer) endpointsToAlias(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	lbDNS := make([]string, len(endpoints))
	for i := range endpoints {
		lbDNS[i] = endpoints[i].Hostname
	}
	zoneIDs, err := a.client.GetCanonicalZoneIDs(lbDNS)
	if err != nil {
		return nil, err
	}
	var rset []*route53.ResourceRecordSet

	for _, ep := range endpoints {
		if loadBalancerZoneID, exist := zoneIDs[ep.Hostname]; exist {
			rset = append(rset, a.endpointToAlias(ep, aws.String(loadBalancerZoneID)))
		} else {
			log.Errorf("Canonical Zone ID for endpoint: %s is not found", ep.Hostname)
		}
	}
	return rset, nil
}

//endpointToAlias convert endpoint to an AWS A Alias record
func (a *awsConsumer) endpointToAlias(ep *pkg.Endpoint, canonicalZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              aws.String(pkg.SanitizeDNSName(ep.Hostname)),
			EvaluateTargetHealth: aws.Bool(evaluateTargetHealth),
			HostedZoneId:         canonicalZoneID,
		},
	}
	return rs
}
