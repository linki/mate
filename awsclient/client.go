package awsclient

import (
	"errors"
	"fmt"
	"time"

	"github.bus.zalan.do/teapot/mate/pkg"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

const (
	defaultSessionDuration = 30 * time.Minute
	defaultTTL             = 300
)

// TODO: move to somewhere
type Logger interface {
	Infoln(...interface{})
}

type defaultLog struct{}

func (l defaultLog) Infoln(args ...interface{}) {
	log.Infoln(args...)
}

type Options struct {
	HostedZone   string
	RecordSetTTL int
	Log          Logger
	GroupID      string
}

type Client struct {
	options Options
}

var ErrInvalidAWSResponse = errors.New("invalid AWS response")

func New(o Options) *Client {
	if o.RecordSetTTL <= 0 {
		o.RecordSetTTL = defaultTTL
	}

	if o.Log == nil {
		o.Log = defaultLog{}
	}

	return &Client{o}
}

//ListRecordSets ...
//retrieve A records filtered by aws group id
func (c *Client) ListRecordSets() ([]*route53.ResourceRecordSet, error) {
	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}
	zoneID, err := c.getZoneID(client)
	if err != nil {
		return nil, err
	}
	if zoneID == nil {
		return nil, fmt.Errorf("hosted zone not found: %s", c.options.HostedZone)
	}
	// TODO: implement paging
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: zoneID,
	}
	rsp, err := client.ListResourceRecordSets(params)
	if err != nil {
		return nil, err
	}

	if rsp == nil {
		return nil, ErrInvalidAWSResponse
	}

	rsets := c.filterByGroupID(rsp.ResourceRecordSets)
	return rsets, nil
}

func (c *Client) ChangeRecordSets(upsert, del []*route53.ResourceRecordSet) error {
	client, err := c.initRoute53Client()
	if err != nil {
		return err
	}

	zoneID, err := c.getZoneID(client)
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, mapChanges("UPSERT", upsert)...)
	changes = append(changes, mapChanges("DELETE", del)...)
	if len(changes) > 0 {
		params := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53.ChangeBatch{
				Changes: changes,
			},
			HostedZoneId: zoneID,
		}
		_, err = client.ChangeResourceRecordSets(params)
		return err
	}
	return nil
}

func (c *Client) MapEndpoints(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	var rset []*route53.ResourceRecordSet
	elbs, err := c.getELBDescriptions(endpoints)
	if err != nil {
		return nil, err
	}
	for _, ep := range endpoints {
		aliasZoneID := getELBZoneID(ep, elbs)
		rset = append(rset, mapEndpointAlias(ep, int64(c.options.RecordSetTTL), aliasZoneID))
		rset = append(rset, mapEndpointTXT(ep, int64(c.options.RecordSetTTL), c.options.GroupID))
	}
	return rset, nil
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
