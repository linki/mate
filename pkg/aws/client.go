package aws

import (
	"errors"
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
)

const (
	defaultSessionDuration = 30 * time.Minute
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
	HostedZone string
	Log        Logger
	GroupID    string
}

type Client struct {
	options Options
}

var ErrInvalidAWSResponse = errors.New("invalid AWS response")

func New(o Options) *Client {

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
	zoneIDs, err := c.getCanonicalZoneIDs(endpoints)
	if err != nil {
		return nil, err
	}
	var rset []*route53.ResourceRecordSet

	for _, ep := range endpoints {
		if LoadBalancerZoneID, exist := zoneIDs[ep.Hostname]; exist {
			rset = append(rset, c.MapEndpointAlias(ep, aws.String(LoadBalancerZoneID)))
			rset = append(rset, c.MapEndpointTXT(ep))
		} else {
			log.Errorf("Canonical Zone ID for endpoint: %s is not found", ep.Hostname)
		}
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
