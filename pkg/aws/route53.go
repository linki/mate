package aws

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
)

var (
	evaluateTargetHealth = true
	defaultTxtTTL        = int64(300)
)

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

//MapEndpointAlias ...
//create an AWS A Alias record
func (c *Client) MapEndpointAlias(ep *pkg.Endpoint, aliasHostedZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              aws.String(ep.Hostname),
			EvaluateTargetHealth: aws.Bool(evaluateTargetHealth),
			HostedZoneId:         aliasHostedZoneID,
		},
	}
	return rs
}

//MapEndpointTXT ...
//create an AWS TXT record
func (c *Client) MapEndpointTXT(ep *pkg.Endpoint) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		TTL:  aws.Int64(defaultTxtTTL),
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(c.GetGroupID()),
		}},
	}
	return rs
}

// mapChanges ...
// create a change batch per action
func mapChanges(action string, rsets []*route53.ResourceRecordSet) []*route53.Change {
	var changes []*route53.Change
	for _, rset := range rsets {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: rset,
		})
	}
	return changes
}

// GetHostedZones ...
// returns the map hosted zone name -> zone id
func (c *Client) GetHostedZones() (map[string]string, error) {
	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}

	output, err := client.ListHostedZones(nil)
	if err != nil {
		return nil, err
	}
	if len(output.HostedZones) == 0 {
		return nil, errors.New("No hosted zones found")
	}

	result := map[string]string{}
	for _, zone := range output.HostedZones {
		result[aws.StringValue(zone.Name)] = aws.StringValue(zone.Id)
	}

	return result, nil
}
