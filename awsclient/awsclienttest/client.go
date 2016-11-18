package awsclienttest

import (
	"errors"
	"fmt"

	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

type Options struct {
	HostedZone   string
	RecordSetTTL int
	GroupID      string
}

type Client struct {
	Records      map[string]string
	AliasRecords map[string]string
	LastUpsert   []*route53.ResourceRecordSet
	LastDelete   []*route53.ResourceRecordSet
	failNext     error
	Options      Options
}

func (c *Client) ListRecordSets() ([]*route53.ResourceRecordSet, error) {
	if err := c.checkFailNext(); err != nil {
		return nil, err
	}

	var records []*pkg.Endpoint

	for dns, ip := range c.Records {
		records = append(records, &pkg.Endpoint{DNSName: dns, IP: ip})
	}

	for dns, hostname := range c.AliasRecords {
		records = append(records, &pkg.Endpoint{DNSName: dns, Hostname: hostname})
	}

	rrecords, err := c.MapEndpoints(records)
	return rrecords, err
}

func (c *Client) MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	var rset []*route53.ResourceRecordSet
	for _, ep := range endpoints {
		aliasZoneID := "test"
		rset = append(rset, mapEndpointAlias(ep, int64(c.Options.RecordSetTTL), &aliasZoneID))
		rset = append(rset, mapEndpointTXT(ep, int64(c.Options.RecordSetTTL), c.Options.GroupID))
	}
	return rset, nil
}

func (c *Client) ChangeRecordSets(upsert, del []*route53.ResourceRecordSet) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}

	c.LastDelete = del
	for _, ep := range del {
		if ep.AliasTarget == nil {
			delete(c.Records, *ep.Name)
		} else {
			delete(c.AliasRecords, *ep.Name)
		}
	}

	c.LastUpsert = upsert
	for _, ep := range upsert {
		if ep.AliasTarget == nil {
			if c.Records == nil {
				c.Records = make(map[string]string)
			}

			c.Records[*ep.Name] = *ep.Name //need to fix this
		} else {
			if c.AliasRecords == nil {
				c.AliasRecords = make(map[string]string)
			}

			c.AliasRecords[*ep.Name] = *ep.AliasTarget.DNSName
		}
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

func mapEndpointAlias(ep *pkg.Endpoint, ttl int64, aliasHostedZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              aws.String(ep.Hostname),
			EvaluateTargetHealth: aws.Bool(true),
			HostedZoneId:         aliasHostedZoneID,
		},
	}
	return rs
}

//mapEndpointTXT ...
//create an AWS TXT record
func mapEndpointTXT(ep *pkg.Endpoint, ttl int64, awsGroupID string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		TTL:  &ttl,
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(getTXTValue(awsGroupID)),
		}},
	}
	return rs
}

func getTXTValue(groupID string) string {
	return fmt.Sprintf("\"mate:%s\"", groupID)
}

func (c *Client) Diff(rset1 []*route53.ResourceRecordSet, rset2 []*route53.ResourceRecordSet) []*route53.ResourceRecordSet {
	var diff []*route53.ResourceRecordSet
	for _, r1 := range rset1 {
		exist := false
		for _, r2 := range rset2 {
			if aws.StringValue(r1.Name) == aws.StringValue(r2.Name) {
				exist = true
				break
			}
		}
		if !exist {
			diff = append(diff, r1)
		}
	}
	return diff
}
