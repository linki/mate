package awsclient

import (
	"time"
	"errors"
	"fmt"
	"strings"

	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

const (
	defaultRole            = "Shibboleth-PowerUser"
	defaultSessionDuration = 30 * time.Minute
	defaultTTL             = 300
)

// TODO: move to somewhere
type Logger interface {
	Infoln(...interface{})
}

type Options struct {
	Role            string
	SessionDuration time.Duration
	RecordSetTTL    int
	HostedZone      string
	Log             Logger
	AccountID       string
}

type Client struct {
	options Options
}

var ErrInvalidAWSResponse = errors.New("invalid AWS response")

func New(o Options) *Client {
	if o.Role == "" {
		o.Role = defaultRole
	}

	if o.SessionDuration <= 0 {
		o.SessionDuration = defaultSessionDuration
	}

	if o.RecordSetTTL <= 0 {
		o.RecordSetTTL = defaultTTL
	}

	return &Client{o}
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	client, err := c.initClient()
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, c.mapEndpoints("UPSERT", upsert)...)
	changes = append(changes, c.mapEndpoints("DELETE", del)...)

	changeSet := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: aws.String(c.options.HostedZone),
	}

	_, err = client.ChangeResourceRecordSets(changeSet)
	return err
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

func (c *Client) initClient() (*route53.Route53, error) {
	// TODO:
	// - is the parent session really needed, or can it be simplified
	// - try to reuse based on the session duration, or, if the error
	// can be identified, refreshing on auth errors only

	parentSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Logger: aws.LoggerFunc(c.options.Log.Infoln),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	credentials := stscreds.NewCredentials(
		parentSession,
		"arn:aws:iam::"+c.options.AccountID+":role/"+c.options.Role,
		func(provider *stscreds.AssumeRoleProvider) {
			provider.Duration = c.options.SessionDuration
			provider.RoleSessionName = "odd-updater@" + c.options.AccountID
		})
	cfg := aws.NewConfig().
		WithCredentialsChainVerboseErrors(true).
		WithCredentials(credentials).
		WithLogger(aws.LoggerFunc(c.options.Log.Infoln))
	session, err := session.NewSession(cfg)
	return nil, err

	return route53.New(session), nil
}

func (c *Client) mapEndpoint(ep *pkg.Endpoint) *route53.ResourceRecordSet {
	ttl := int64(c.options.RecordSetTTL)
	return &route53.ResourceRecordSet{
		Name: aws.String(ep.DNSName),
		Type: aws.String("A"),
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(ep.IP),
		}},
		TTL: &ttl,
	}
}

func (c *Client) mapEndpoints(action string, endpoints []*pkg.Endpoint) []*route53.Change {
	var changes []*route53.Change
	for _, ep := range endpoints {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: c.mapEndpoint(ep),
		})
	}

	return changes
}

func (c *Client) getZoneID(ac *route53.Route53) (*string, error) {
	// TODO: handle when not all hosted zones fit in the response
	zonesResult, err := ac.ListHostedZones(nil);
	if err != nil {
		return nil, err
	}

	if zonesResult == nil {
		return nil, ErrInvalidAWSResponse
	}

	zoneName := c.options.HostedZone
	if !strings.HasSuffix(zoneName, ".") {
		zoneName += "."
	}

	var zoneID *string
	for _, z := range zonesResult.HostedZones {
		if aws.StringValue(z.Name) == zoneName {
			zoneID = z.Id
			break
		}
	}

	return zoneID, nil
}

func mapRecordSets(sets []*route53.ResourceRecordSet) []*pkg.Endpoint {
	var endpoints []*pkg.Endpoint
	for _, s := range sets {
		if aws.StringValue(s.Type) != "A" {
			continue
		}

		var ip string
		for _, r := range s.ResourceRecords {
			ip = aws.StringValue(r.Value)
		}

		endpoints = append(endpoints, &pkg.Endpoint{DNSName: aws.StringValue(s.Name), IP: ip})
	}

	return endpoints
}
