package consumers

import (
	"errors"

	"gopkg.in/alecthomas/kingpin.v2"

	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

// Implementations provide access to AWS Route53 API's
// required calls.
type AWSClient interface {
	ListRecordSets() ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet) error
	EndpointsToAlias(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error)
	RecordInfo(records []*route53.ResourceRecordSet) map[string]*pkg.RecordInfo
	GetGroupID() string
	GetAssignedTXTRecordObject(record *route53.ResourceRecordSet) *route53.ResourceRecordSet
}

type awsClient struct {
	client AWSClient
}

func init() {
	kingpin.Flag("aws-hosted-zone", "The hosted zone name for the AWS consumer (required with AWS).").StringVar(&params.awsHostedZone)
	kingpin.Flag("aws-record-group-id", "Identifier to filter the mate records ").StringVar(&params.awsGroupID)
}

// NewAWSRoute53 reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSRoute53() (Consumer, error) {
	if params.awsHostedZone == "" {
		return nil, errors.New("please provide --aws-hosted-zone")
	}
	if params.awsGroupID == "" {
		return nil, errors.New("please provide --aws-record-group-id")
	}
	return withClient(awsclient.New(awsclient.Options{
		HostedZone: params.awsHostedZone,
		GroupID:    params.awsGroupID,
	})), nil
}

func withClient(c AWSClient) *awsClient {
	return &awsClient{c}
}

func (a *awsClient) Sync(endpoints []*pkg.Endpoint) error {
	existingRecords, err := a.client.ListRecordSets()
	if err != nil {
		return err
	}

	recordInfoMap := a.client.RecordInfo(existingRecords)
	newAliasRecords, err := a.client.EndpointsToAlias(endpoints)
	if err != nil {
		return err
	}

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
		if existingRecordInfo.Target != aws.StringValue(newAliasRecord.AliasTarget.DNSName) {
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
		return a.client.ChangeRecordSets(upsert, del, nil)
	}

	log.Infoln("No changes submitted")

	return nil
}

func (a *awsClient) Process(endpoint *pkg.Endpoint) error {
	aliasRecords, err := a.client.EndpointsToAlias([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}

	create := []*route53.ResourceRecordSet{aliasRecords[0], a.client.GetAssignedTXTRecordObject(aliasRecords[0])}

	err = a.client.ChangeRecordSets(nil, nil, create)
	if err != nil && strings.Contains(err.Error(), "it already exists") {
		log.Warnf("Record [name=%s] could not be created, another record with same name already exists", endpoint.DNSName)
		return nil
	}
	return err
}
