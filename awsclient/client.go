package awsclient

import (
	"errors"
	"strings"
	"time"

	"github.bus.zalan.do/teapot/mate/pkg"
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
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

//ListMateRecordSets ...
//retrieve all records (A records + TXT) created by mate and convert into endpoints
func (c *Client) ListMateRecordSets(clusterName string) ([]*pkg.Endpoint, error) {
	rs, err := c.getRecordSets()
	if err != nil {
		return nil, err
	}
	frs := filterMate(rs, clusterName)
	eps := mapRecordSets(frs)
	if err != nil {
		return nil, err
	}
	return eps, nil
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	err := c.attachELBZoneIDs(upsert)
	if err != nil {
		return err
	}
	client, err := c.initRoute53Client()
	if err != nil {
		return err
	}

	zoneID, err := c.getZoneID(client)
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, c.actionRecords("UPSERT", zoneID, upsert)...)
	changes = append(changes, c.actionRecords("DELETE", zoneID, del)...)
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

func (c *Client) initRoute53Client() (*route53.Route53, error) {
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

func (c *Client) initELBClient() (*elb.ELB, error) {
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
	return elb.New(session), nil
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

		hostname := s.AliasTarget.DNSName
		endpoints = append(endpoints, &pkg.Endpoint{
			DNSName:     aws.StringValue(s.Name),
			Hostname:    *hostname,
			AliasZoneID: *s.AliasTarget.HostedZoneId,
		})
	}
	return endpoints
}
