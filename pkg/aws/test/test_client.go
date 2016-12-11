package test

import (
	"errors"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
)

type Options struct {
	HostedZone   string
	RecordSetTTL int
	GroupID      string
}

type Result struct {
	LastUpsert []*route53.ResourceRecordSet
	LastDelete []*route53.ResourceRecordSet
	LastCreate []*route53.ResourceRecordSet
}

type Client struct {
	Current  map[string][]*route53.ResourceRecordSet
	Output   map[string]Result
	Zones    map[string]string
	failNext error
	Options  Options
	*awsclient.Client
}

func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
	return c.Current[zoneID], nil
}

func (c *Client) MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	var rset []*route53.ResourceRecordSet
	aliasZoneID := "test"
	for _, ep := range endpoints {
		rset = append(rset, c.Client.MapEndpointAlias(ep, &aliasZoneID))
		rset = append(rset, c.Client.MapEndpointTXT(ep))
	}
	return rset, nil
}

func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}
	if c.Output == nil {
		c.Output = map[string]Result{}
	}
	c.Output[zoneID] = Result{
		LastCreate: create,
		LastDelete: del,
		LastUpsert: upsert,
	}
	return nil
}

func (c *Client) FailNext() {
	c.failNext = errors.New("test error")
}

func (c *Client) checkFailNext() (err error) {
	err, c.failNext = c.failNext, nil
	return
}

func (c *Client) GetHostedZones() (map[string]string, error) {
	return c.Zones, nil
}
