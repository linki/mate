package awsclient

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.bus.zalan.do/teapot/mate/pkg"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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

func (c *Client) ListRecordSets() ([]*pkg.Endpoint, error) {
	client, err := c.initClient()
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

	return mapRecordSets(rsp.ResourceRecordSets), nil
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	client, err := c.initClient()
	if err != nil {
		return err
	}

	zoneID, err := c.getZoneID(client)
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, c.mapEndpoints("UPSERT", upsert, zoneID)...)
	changes = append(changes, c.mapEndpoints("DELETE", del, zoneID)...)
	if len(changes) == 0 {
		return nil
	}

	changeSet := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: zoneID,
	}

	_, err = client.ChangeResourceRecordSets(changeSet)
	return err
}

func (c *Client) initClient() (*route53.Route53, error) {
	session, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Logger: aws.LoggerFunc(c.options.Log.Infoln),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	return route53.New(session), nil
}

func (c *Client) getZoneID(ac *route53.Route53) (*string, error) {
	// TODO: handle when not all hosted zones fit in the response
	zonesResult, err := ac.ListHostedZones(nil)
	if err != nil {
		return nil, err
	}

	if zonesResult == nil {
		return nil, ErrInvalidAWSResponse
	}

	zoneName := hostedZoneSuffix(c.options.HostedZone)

	var zoneID *string
	for _, z := range zonesResult.HostedZones {
		if aws.StringValue(z.Name) == zoneName {
			zoneID = z.Id
			break
		}
	}

	return zoneID, nil
}

func cleanHostedZoneID(zoneID *string) *string {
	v := *zoneID
	qualifierLength := strings.LastIndex(v, "/")
	if qualifierLength < 0 {
		return &v
	}

	v = v[qualifierLength+1:]
	return &v
}

func hostedZoneSuffix(name string) string {
	if !strings.HasSuffix(name, ".") {
		return name + "."
	}

	return name
}

func mapRecordSets(sets []*route53.ResourceRecordSet) []*pkg.Endpoint {
	var endpoints []*pkg.Endpoint
	for _, s := range sets {
		if aws.StringValue(s.Type) != "A" {
			continue
		}

		var ip, hostname string
		if s.AliasTarget != nil {
			hostname = aws.StringValue(s.AliasTarget.DNSName)
		} else {
			for _, r := range s.ResourceRecords {
				ip = aws.StringValue(r.Value)
				break
			}
		}

		endpoints = append(endpoints, &pkg.Endpoint{
			DNSName:  aws.StringValue(s.Name),
			IP:       ip,
			Hostname: hostname,
		})
	}

	return endpoints
}

func (c *Client) mapEndpoint(ep *pkg.Endpoint, aliasHostedZoneId *string) *route53.ResourceRecordSet {
	ttl := int64(c.options.RecordSetTTL)
	vfalse := false
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(ep.DNSName),
	}

	if ep.IP != "" {
		rs.TTL = &ttl
		rs.ResourceRecords = []*route53.ResourceRecord{{
			Value: aws.String(ep.IP),
		}}
	} else {
		rs.AliasTarget = &route53.AliasTarget{
			DNSName:              aws.String(ep.Hostname),
			HostedZoneId:         cleanHostedZoneID(aliasHostedZoneId),
			EvaluateTargetHealth: &vfalse,
		}
	}

	return rs
}

// AWS alias records have a required field expecting the hosted zone id.
// Tested only with the AWS Console, aliases to load balancers don't work
// across hosted zones, but API expects this field, so let them have it.
func (c *Client) mapEndpoints(action string, endpoints []*pkg.Endpoint, aliasHostedZoneId *string) []*route53.Change {
	var changes []*route53.Change
	for _, ep := range endpoints {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: c.mapEndpoint(ep, aliasHostedZoneId),
		})
	}

	return changes
}
