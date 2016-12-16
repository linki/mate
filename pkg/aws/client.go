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

//EndpointsToAlias converts pkg Endpoint to route53 Alias Records
func (c *Client) EndpointsToAlias(endpoints []*pkg.Endpoint) ([]*route53.ResourceRecordSet, error) {
	zoneIDs, err := c.getCanonicalZoneIDs(endpoints)
	if err != nil {
		return nil, err
	}
	var rset []*route53.ResourceRecordSet

	for _, ep := range endpoints {
		if loadBalancerZoneID, exist := zoneIDs[ep.Hostname]; exist {
			rset = append(rset, c.EndpointToAlias(ep, aws.String(loadBalancerZoneID)))
		} else {
			log.Errorf("Canonical Zone ID for endpoint: %s is not found", ep.Hostname)
		}
	}
	return rset, nil
}

//RecordInfo returns the map of record assigned dns to its target and groupID (can be empty)
func (c *Client) RecordInfo(records []*route53.ResourceRecordSet) map[string]*pkg.RecordInfo {
	groupIDMap := map[string]string{} //maps dns to group ID

	for _, record := range records {
		if (aws.StringValue(record.Type)) == "TXT" {
			groupIDMap[aws.StringValue(record.Name)] = aws.StringValue(record.ResourceRecords[0].Value)
		} else {
			if _, exist := groupIDMap[aws.StringValue(record.Name)]; !exist {
				groupIDMap[aws.StringValue(record.Name)] = ""
			}
		}
	}

	infoMap := map[string]*pkg.RecordInfo{}
	for _, record := range records {
		groupID := groupIDMap[aws.StringValue(record.Name)]
		if _, exist := infoMap[aws.StringValue(record.Name)]; !exist {
			infoMap[aws.StringValue(record.Name)] = &pkg.RecordInfo{
				GroupID: groupID,
			}
		}
		if aws.StringValue(record.Type) != "TXT" {
			infoMap[aws.StringValue(record.Name)].Target = getRecordTarget(record)
		}
	}

	return infoMap
}

//GetGroupID returns the idenitifier for AWS records as stored in TXT records
func (c *Client) GetGroupID() string {
	return fmt.Sprintf("\"mate:%s\"", c.options.GroupID)
}

//GetAssignedTXTRecordObject returns the TXT record which accompanies the Alias record
func (c *Client) GetAssignedTXTRecordObject(r *route53.ResourceRecordSet) *route53.ResourceRecordSet {
	return c.createTXTRecordObject(aws.StringValue(r.Name))
}

func getRecordTarget(r *route53.ResourceRecordSet) string {
	if r.AliasTarget != nil {
		return aws.StringValue(r.AliasTarget.DNSName)
	}
	return aws.StringValue(r.ResourceRecords[0].Value)
}
