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
	ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error
	MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error)
	RecordMap(records []*route53.ResourceRecordSet) map[string]string
	GetGroupID() string
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
	next, err := a.client.MapEndpoints(endpoints)
	if err != nil {
		return err
	}

	isProcessed := make([]bool, len(next), len(next)) //indicates whether the record was already processed

	hostedZonesMap, err := a.client.GetHostedZones()

	if err != nil {
		return err
	}

	for zoneName, zoneID := range hostedZonesMap {
		records, err := a.client.ListRecordSets(zoneID)
		if err != nil {
			return err
		}

		recordMap := a.client.RecordMap(records)

		var upsert, del []*route53.ResourceRecordSet

		for i, endpoint := range next {
			groupID, exist := recordMap[aws.StringValue(endpoint.Name)]

			if exist && groupID != a.client.GetGroupID() { //if the record exists, but does not have associated TXT record or TXT record group id is different
				log.Warnf("Skipping record %s: with a group ID: %s", aws.StringValue(endpoint.Name), groupID)
				continue
			}

			if !exist && strings.HasSuffix(aws.StringValue(endpoint.Name), zoneName) && !isProcessed[i] {
				upsert = append(upsert, endpoint)
				isProcessed[i] = true
			}
			if exist && groupID == a.client.GetGroupID() && !isProcessed[i] { //TODO:ideahitme: fix the  unnecessary upsert
				upsert = append(upsert, endpoint)
				isProcessed[i] = true
			}
		}

		for _, record := range records {
			groupID := recordMap[aws.StringValue(record.Name)]
			if groupID == a.client.GetGroupID() {
				remove := true
				for _, endpoint := range next {
					if aws.StringValue(endpoint.Name) == aws.StringValue(record.Name) {
						remove = false
					}
				}
				if remove {
					del = append(del, record)
				}
			}
		}
		if len(upsert) > 0 || len(del) > 0 {
			err := a.client.ChangeRecordSets(upsert, del, nil, zoneID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *awsClient) Process(endpoint *pkg.Endpoint) error {
	hostedZonesMap, err := a.client.GetHostedZones()
	if err != nil {
		return err
	}

	create, err := a.client.MapEndpoints([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}

	for zoneName, zoneID := range hostedZonesMap {
		if !strings.HasSuffix(aws.StringValue(create[0].Name), zoneName) { // speicified DNS does not match the hosted zones domain
			continue
		}
		err = a.client.ChangeRecordSets(nil, nil, create, zoneID)
		if err != nil && strings.Contains(err.Error(), "already exists") {
			log.Warnf("Record [name=%s] could not be created, another record with same name already exists", endpoint.DNSName)
			return nil
		}
		return err
	}

	log.Infoln("Hosted zone not found. Skipping record: ", endpoint.DNSName)
	return nil
}
