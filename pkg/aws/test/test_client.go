package test

import (
	"errors"

	"github.bus.zalan.do/teapot/mate/pkg"
	awsclient "github.bus.zalan.do/teapot/mate/pkg/aws"
	"github.com/aws/aws-sdk-go/service/route53"
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

func (c *Client) MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	var rset []*route53.ResourceRecordSet
	aliasZoneID := "test"
	for _, ep := range endpoints {
		rset = append(rset, c.Client.MapEndpointAlias(ep, &aliasZoneID))
		rset = append(rset, c.Client.MapEndpointTXT(ep))
	}
	return rset, nil
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
