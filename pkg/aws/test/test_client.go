package test

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

type Options struct {
	HostedZone   string
	RecordSetTTL int
	GroupID      string
}

type Client struct {
	Current    []*route53.ResourceRecordSet
	LastUpsert []*route53.ResourceRecordSet
	LastDelete []*route53.ResourceRecordSet
	LastCreate []*route53.ResourceRecordSet
	failNext   error
	Options    Options
	*awsclient.Client
}

func (c *Client) ListRecordSets() ([]*route53.ResourceRecordSet, error) {
	return c.Current, nil
}

func (c *Client) EndpointsToAlias(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	var rset []*route53.ResourceRecordSet
	canonicalZoneID := "test"
	for _, ep := range endpoints {
		rset = append(rset, c.endpointToAlias(ep, &canonicalZoneID))
	}
	return rset, nil
}

func (c *Client) endpointToAlias(ep *pkg.Endpoint, canonicalZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              aws.String(pkg.SanitizeDNSName(ep.Hostname)),
			EvaluateTargetHealth: aws.Bool(false),
			HostedZoneId:         canonicalZoneID,
		},
	}
	return rs
}

func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}

	c.LastCreate = create
	c.LastDelete = del
	c.LastUpsert = upsert

	return nil
}

func (c *Client) FailNext() {
	c.failNext = errors.New("test error")
}

func (c *Client) checkFailNext() (err error) {
	err, c.failNext = c.failNext, nil
	return
}
