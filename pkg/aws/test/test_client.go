package test

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

type Options struct {
	GroupID string
}

type Client struct {
	Current    map[string][]*route53.ResourceRecordSet
	LastUpsert map[string][]*route53.ResourceRecordSet
	LastDelete map[string][]*route53.ResourceRecordSet
	LastCreate map[string][]*route53.ResourceRecordSet
	failNext   error
	Options    Options
	*awsclient.Client
}

func NewClient(groupID string) *Client {
	return &Client{
		Client: awsclient.New(awsclient.Options{
			GroupID: groupID,
		}),
		Options: Options{
			GroupID: groupID,
		},
		LastCreate: map[string][]*route53.ResourceRecordSet{},
		LastDelete: map[string][]*route53.ResourceRecordSet{},
		LastUpsert: map[string][]*route53.ResourceRecordSet{},
	}
}

func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
	c.Current = getOriginalState(c.Client.GetGroupID())
	return c.Current[zoneID], nil
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

func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}

	c.LastCreate[zoneID] = create
	c.LastDelete[zoneID] = del
	c.LastUpsert[zoneID] = upsert

	return nil
}

func (c *Client) GetHostedZones() (map[string]string, error) {
	return getHostedZones(), nil
}

func (c *Client) FailNext() {
	c.failNext = errors.New("test error")
}

func (c *Client) checkFailNext() (err error) {
	err, c.failNext = c.failNext, nil
	return
}
