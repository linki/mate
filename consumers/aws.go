package consumers

import (
	"errors"
	"fmt"

	"strings"

	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

// AWSClient interface
type AWSClient interface {
	ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error
	GetCanonicalZoneIDs(lbDNS []string) (map[string]string, error) //get hosted zone ids for the LBs
	GetHostedZones() (map[string]string, error)                    //get all route53 hosted zones for the account
}

type awsConsumer struct {
	groupID string
	client  AWSClient
}

const (
	evaluateTargetHealth = true
	defaultTxtTTL        = int64(300)
	defaultATTL          = int64(300)
)

// NewAWSRoute53Consumer reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSRoute53Consumer(awsRecordGroupID string) (Consumer, error) {
	if awsRecordGroupID == "" {
		return nil, errors.New("please provide --aws-record-group-id")
	}
	return withClient(awsclient.New(awsclient.Options{}), awsRecordGroupID), nil
}

func withClient(c AWSClient, groupID string) *awsConsumer {
	return &awsConsumer{
		groupID: groupID,
		client:  c,
	}
}

func (a *awsConsumer) Sync(endpoints []*pkg.Endpoint) error {
	kubeRecords, err := a.endpointsToRecords(endpoints)
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
	for _, record := range kubeRecords {
		zoneID := getZoneIDForEndpoint(hostedZonesMap, record) //this guarantees that the endpoint will not be created in multiple hosted zones
		if zoneID == "" {
			log.Warnf("Hosted zone for endpoint: %s was not found. Skipping record...", aws.StringValue(record.Name))
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
				log.Errorf("Error changing records per zone: %s. Error: %v", zoneName, err)
			}
		}(zoneName, zoneID)
	}
	wg.Wait()
	return nil
}

func (a *awsConsumer) syncPerHostedZone(kubeRecords []*route53.ResourceRecordSet, zoneID string) error {
	existingRecords, err := a.client.ListRecordSets(zoneID)
	if err != nil {
		log.Debugf("aborting sync per hosted zone. Cannot convert endpoints to rrs: %v", err)
		return err
	}

	recordInfoMap := a.recordInfo(existingRecords)

	var upsert, del []*route53.ResourceRecordSet
	upsertedMap := make(map[string]bool) // keep track of records to be upserted
	targetMap := map[string][]*string{}  // map dnsname -> list of targets
	for _, kr := range kubeRecords {
		targetMap[aws.StringValue(kr.Name)] = append(targetMap[aws.StringValue(kr.Name)], aws.String(a.getRecordTarget(kr)))
	}
	//find records to be upserted
	for _, kubeRecord := range kubeRecords {
		//make sure that another record with same DNS name was not already included into upsert slice
		if _, upserted := upsertedMap[aws.StringValue(kubeRecord.Name)]; upserted {
			continue
		}

		existingRecordInfo, exist := recordInfoMap[aws.StringValue(kubeRecord.Name)]

		if !exist { //record does not exist, create it
			newTXTRecord := a.getAssignedTXTRecordObject(kubeRecord)
			upsert = append(upsert, kubeRecord, newTXTRecord)
			upsertedMap[aws.StringValue(kubeRecord.Name)] = true
			continue
		}

		if existingRecordInfo.GroupID != a.getGroupID() { // there exist a record with a different or empty group ID
			log.Warnf("Skipping record %s: with a group ID: %s", aws.StringValue(kubeRecord.Name), existingRecordInfo.GroupID)
			continue
		}

		//there exists a record in AWS Route53 with same DNS name and group id, but need to make sure that
		//the alias load balancer is no longer used
		kubeTargetsForDNS := targetMap[aws.StringValue(kubeRecord.Name)]
		targetStillRequired := false
		for _, targetPtr := range kubeTargetsForDNS {
			if pkg.SameDNSName(aws.StringValue(targetPtr), existingRecordInfo.Target) {
				targetStillRequired = true
				break
			}
		}
		if !targetStillRequired { //target is no longer required - overwrite it
			newTXTRecord := a.getAssignedTXTRecordObject(kubeRecord)
			upsert = append(upsert, kubeRecord, newTXTRecord)
			upsertedMap[aws.StringValue(kubeRecord.Name)] = true
		}
	}

	//find records to be removed
	for _, existingRecord := range existingRecords {
		recordInfo := recordInfoMap[aws.StringValue(existingRecord.Name)]
		if recordInfo.GroupID == a.getGroupID() {
			remove := true
			for _, kubeRecord := range kubeRecords {
				if pkg.SameDNSName(aws.StringValue(kubeRecord.Name), aws.StringValue(existingRecord.Name)) {
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

	ARecords, err := a.endpointsToRecords([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}
	if len(ARecords) != 1 {
		return fmt.Errorf("failed to process endpoint. A record could not be constructed for: %s:%s:%s", endpoint.DNSName, endpoint.Hostname, endpoint.IP)
	}

	create := []*route53.ResourceRecordSet{ARecords[0], a.getAssignedTXTRecordObject(ARecords[0])}

	zoneID := getZoneIDForEndpoint(hostedZonesMap, ARecords[0])
	if zoneID == "" {
		log.Warnf("Hosted zone for endpoint: %s was not found. Skipping record...", endpoint.DNSName)
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

//groupIDInfo builds a map from dns name to its group ID
func (a *awsConsumer) groupIDInfo(records []*route53.ResourceRecordSet) map[string]string {
	groupIDMap := map[string]string{} //maps dns to group ID

	for _, record := range records {
		if aws.StringValue(record.Type) == "TXT" {
			if len(record.ResourceRecords) > 0 {
				groupIDMap[aws.StringValue(record.Name)] = aws.StringValue(record.ResourceRecords[0].Value)
			} else {
				log.Errorf("Unexpected response from AWS API, got TXT record with empty resources: %s. Record is excluded from syncing", aws.StringValue(record.Name))
				groupIDMap[aws.StringValue(record.Name)] = ""
			}
		} else {
			if _, exist := groupIDMap[aws.StringValue(record.Name)]; !exist {
				groupIDMap[aws.StringValue(record.Name)] = ""
			}
		}
	}
	return groupIDMap
}

//recordInfo returns the map of record assigned dns to its target and groupID (can be empty string)
func (a *awsConsumer) recordInfo(records []*route53.ResourceRecordSet) map[string]*pkg.RecordInfo {
	groupIDMap := a.groupIDInfo(records)
	infoMap := map[string]*pkg.RecordInfo{} //maps record DNS to its GroupID (if exists) and Target (LB)
	for _, record := range records {
		groupID := groupIDMap[aws.StringValue(record.Name)]
		if _, exist := infoMap[aws.StringValue(record.Name)]; !exist {
			infoMap[aws.StringValue(record.Name)] = &pkg.RecordInfo{
				GroupID: groupID,
			}
		}
		if aws.StringValue(record.Type) != "TXT" {
			infoMap[aws.StringValue(record.Name)].Target = a.getRecordTarget(record) //sanitization not needed here, as per IP case
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

//endpointsToRecords converts pkg Endpoint to route53 A [Alias] Records depending whether IP/LB Hostname is used
func (a *awsConsumer) endpointsToRecords(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	lbDNS := make([]string, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.Hostname != "" {
			lbDNS = append(lbDNS, endpoint.Hostname)
		}
	}
	zoneIDs, err := a.client.GetCanonicalZoneIDs(lbDNS)
	if err != nil {
		return nil, err
	}
	var rset []*route53.ResourceRecordSet

	for _, ep := range endpoints {
		if loadBalancerZoneID, exist := zoneIDs[ep.Hostname]; exist {
			rset = append(rset, a.endpointToRecord(ep, aws.String(loadBalancerZoneID)))
		} else if ep.IP != "" {
			rset = append(rset, a.endpointToRecord(ep, aws.String("")))
		} else {
			return nil, fmt.Errorf("Canonical Zone ID for load balancer: %s was not found", ep.Hostname)
		}
	}
	return rset, nil
}

//endpointToRecord convert endpoint to an AWS A [Alias] record depending whether IP of LB hostname is used
//if both are specified hostname takes precedence and Alias record is to be created
func (a *awsConsumer) endpointToRecord(ep *pkg.Endpoint, canonicalZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
	}
	if ep.Hostname != "" {
		rs.AliasTarget = &route53.AliasTarget{
			DNSName:              aws.String(pkg.SanitizeDNSName(ep.Hostname)),
			EvaluateTargetHealth: aws.Bool(evaluateTargetHealth),
			HostedZoneId:         canonicalZoneID,
		}
	} else {
		rs.TTL = aws.Int64(defaultATTL)
		rs.ResourceRecords = []*route53.ResourceRecord{
			&route53.ResourceRecord{
				Value: aws.String(ep.IP),
			},
		}
	}
	return rs
}
