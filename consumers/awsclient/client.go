package awsclient

import (
	"time"

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

	changeSet := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{},
		HostedZoneId: aws.String(c.options.HostedZone),
	}

	changeSet.ChangeBatch.Changes = append(changeSet.ChangeBatch.Changes, c.mapChanges("UPSERT", upsert)...)
	changeSet.ChangeBatch.Changes = append(changeSet.ChangeBatch.Changes, c.mapChanges("DELETE", del)...)

	_, err = client.ChangeResourceRecordSets(changeSet)
	return err
}

func (c *Client) initClient() (*route53.Route53, error) {
	// TODO:
	// - is the parent session really needed, or can it be simplified
	// - try to reuse based on the session duration

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

func (c *Client) mapRecord(ep *pkg.Endpoint) *route53.ResourceRecordSet {
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

func (c *Client) mapChanges(action string, endpoints []*pkg.Endpoint) []*route53.Change {
	var changes []*route53.Change
	for _, ep := range endpoints {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: c.mapRecord(ep),
		})
	}

	return changes
}
