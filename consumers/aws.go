package consumers

import (
	"errors"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/awsclient"
	"github.bus.zalan.do/teapot/mate/pkg"
)

// Implementations provide access to AWS Route53 API's
// required calls.
type AWSClient interface {
	ListMateRecordSets(clusterName string) ([]*pkg.Endpoint, error)
	ChangeRecordSets(upsert, del []*pkg.Endpoint) error
}

type aws struct {
	client AWSClient
}

func init() {
	kingpin.Flag("aws-hosted-zone", "The hosted zone name for the AWS consumer (required with AWS).").StringVar(&params.awsHostedZone)
	kingpin.Flag("aws-record-set-ttl", "TTL for the record sets created by the AWS consumer.").IntVar(&params.awsTTL)
}

// NewAWS reates a Consumer instance to sync and process DNS
// entries in AWS Route53.
func NewAWSRoute53() (Consumer, error) {
	if params.awsHostedZone == "" {
		return nil, errors.New("please provide --aws-hosted-zone")
	}

	return withClient(awsclient.New(awsclient.Options{
		HostedZone:   params.awsHostedZone,
		RecordSetTTL: params.awsTTL,
	})), nil
}

func withClient(c AWSClient) *aws {
	return &aws{c}
}

func (a *aws) Sync(endpoints []*pkg.Endpoint, clusterName string) error {
	current, err := a.client.ListMateRecordSets(clusterName)
	if err != nil {
		return err
	}

	var upsert, del []*pkg.Endpoint

	for _, ep := range endpoints {
		if needsUpsert(ep, current) {
			upsert = append(upsert, ep)
		}
	}

	for _, ep := range current {
		if needsDelete(ep, endpoints) {
			del = append(del, ep)
		}
	}

	if len(upsert) > 0 || len(del) > 0 {
		return a.client.ChangeRecordSets(upsert, del)
	}

	return nil
}

func (a *aws) Process(endpoint *pkg.Endpoint) error {
	return a.client.ChangeRecordSets([]*pkg.Endpoint{endpoint}, nil)
}

func needsUpsert(ep *pkg.Endpoint, currentEndpoints []*pkg.Endpoint) bool {
	for _, cep := range currentEndpoints {
		if pkg.FQDN(cep.DNSName) == pkg.FQDN(ep.DNSName) {
			return cep.IP != ep.IP || cep.Hostname != ep.Hostname
		}
	}

	return true
}

func needsDelete(ep *pkg.Endpoint, nextEndpoints []*pkg.Endpoint) bool {
	for _, nep := range nextEndpoints {
		if pkg.FQDN(nep.DNSName) == pkg.FQDN(ep.DNSName) {
			return false
		}
	}

	return true
}
