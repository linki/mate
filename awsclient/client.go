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

	return rsp.ResourceRecordSets, nil
}

func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet) error {
	client, err := c.initRoute53Client()
	if err != nil {
		return err
	}

	zoneID, err := c.getZoneID(client)
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, mapChanges("CREATE", create)...)
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
	elbs, err := c.getELBDescriptions(endpoints)
	if err != nil {
		return nil, err
	}

	var rset []*route53.ResourceRecordSet

	for _, ep := range endpoints {
		aliasZoneID := getELBZoneID(ep, elbs)
		rset = append(rset, c.MapEndpointAlias(ep, int64(c.options.RecordSetTTL), aliasZoneID))
		rset = append(rset, c.MapEndpointTXT(ep, int64(c.options.RecordSetTTL)))
	}
	return rset, nil
}

func (c *Client) RecordMap(records []*route53.ResourceRecordSet) map[string]string {
	recordMap := make(map[string]string)

	for _, record := range records {
		if (aws.StringValue(record.Type)) == "TXT" {
			recordMap[aws.StringValue(record.Name)] = aws.StringValue(record.ResourceRecords[0].Value)
		} else {
			if _, exist := recordMap[aws.StringValue(record.Name)]; !exist {
				recordMap[aws.StringValue(record.Name)] = ""
			}
		}
	}

	return recordMap
}

//return the Value of the TXT record as stored
func (c *Client) GetGroupID() string {
	return fmt.Sprintf("\"mate:%s\"", c.options.GroupID)
}
