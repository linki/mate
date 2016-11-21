package consumers

import (
	"errors"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/awsclient"
	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/service/route53"
)

// Implementations provide access to AWS Route53 API's
// required calls.
type AWSClient interface {
	ListRecordSets() ([]*route53.ResourceRecordSet, error)
	ChangeRecordSets(upsert, del []*route53.ResourceRecordSet) error
	MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error)
	Diff(rset1 []*route53.ResourceRecordSet, rset2 []*route53.ResourceRecordSet) []*route53.ResourceRecordSet
}

type aws struct {
	client AWSClient
}

func init() {
	kingpin.Flag("aws-hosted-zone", "The hosted zone name for the AWS consumer (required with AWS).").StringVar(&params.awsHostedZone)
	kingpin.Flag("aws-record-set-ttl", "TTL for the record sets created by the AWS consumer.").IntVar(&params.awsTTL)
	kingpin.Flag("aws-record-group-id", "Identifier to filter the mate records ").StringVar(&params.awsGroupID)
}

// NewAWS reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSRoute53() (Consumer, error) {
	if params.awsHostedZone == "" {
		return nil, errors.New("please provide --aws-hosted-zone")
	}
	if params.awsGroupID == "" {
		return nil, errors.New("please provide --aws-record-group-id")
	}
	return withClient(awsclient.New(awsclient.Options{
		HostedZone:   params.awsHostedZone,
		RecordSetTTL: params.awsTTL,
		GroupID:      params.awsGroupID,
	})), nil
}

func withClient(c AWSClient) *aws {
	return &aws{c}
}

func (a *aws) Sync(endpoints []*pkg.Endpoint) error {
	current, err := a.client.ListRecordSets()
	if err != nil {
		return err
	}
	next, err := a.client.MapEndpoints(endpoints)
	if err != nil {
		return err
	}

	upsert := next
	del := a.client.Diff(current, next)
	if len(upsert) > 0 || len(del) > 0 {
		return a.client.ChangeRecordSets(upsert, del)
	}
	return nil
}

func (a *aws) Process(endpoint *pkg.Endpoint) error {
	upsert, err := a.client.MapEndpoints([]*pkg.Endpoint{endpoint})
	if err != nil {
		return err
	}
	return a.client.ChangeRecordSets(upsert, nil)
}
