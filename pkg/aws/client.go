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
	Log     Logger
	GroupID string
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

//ListRecordSets retrieve all records existing in the specified hosted zone
func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
	records := make([]*route53.ResourceRecordSet, 0)

	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	for {
		rsp, err := client.ListResourceRecordSets(params)
		if err != nil {
			return nil, err
		}

		if rsp == nil {
			log.Warnln("Empty response from AWS ListRecordSets API")
			break
		}

		records = append(records, rsp.ResourceRecordSets...)
		log.Debugf("Page of records per %s. Size: %d. More records: %v", zoneID,
			len(rsp.ResourceRecordSets), aws.BoolValue(rsp.IsTruncated))

		//retrieve next set of records if any
		if !aws.BoolValue(rsp.IsTruncated) {
			break //no more records
		}
		params.SetStartRecordName(aws.StringValue(rsp.NextRecordName))
	}
	return records, nil
}

//ChangeRecordSets creates and submits the record set change against the AWS API
func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error {
	client, err := c.initRoute53Client()
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, createChangesList("CREATE", create)...)
	changes = append(changes, createChangesList("UPSERT", upsert)...)
	changes = append(changes, createChangesList("DELETE", del)...)
	if len(changes) > 0 {
		params := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53.ChangeBatch{
				Changes: changes,
			},
			HostedZoneId: aws.String(zoneID),
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
			rset = append(rset, c.endpointToAlias(ep, aws.String(loadBalancerZoneID)))
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

	infoMap := map[string]*pkg.RecordInfo{} //maps record DNS to its GroupID (if exists) and Target (LB)
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

// GetHostedZones returns the map hosted zone domain name -> zone id
func (c *Client) GetHostedZones() (map[string]string, error) {
	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}

	output, err := client.ListHostedZones(nil)
	if err != nil {
		return nil, err
	}

	hostedZoneMap := map[string]string{}
	for _, zone := range output.HostedZones {
		hostedZoneMap[aws.StringValue(zone.Name)] = aws.StringValue(zone.Id)
	}

	return hostedZoneMap, nil
}
