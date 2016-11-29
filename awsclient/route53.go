package awsclient

import (
	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	EvaluateTargetHealth = true
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
			EvaluateTargetHealth: aws.Bool(EvaluateTargetHealth),
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

func (c *Client) getZoneID(ac *route53.Route53) (*string, error) {
	zonesResult, err := ac.ListHostedZones(nil)
	if err != nil {
		return nil, err
	}

	if zonesResult == nil {
		return nil, ErrInvalidAWSResponse
	}

	zoneName := pkg.SanitizeDNSName(c.options.HostedZone)

	var zoneID *string
	for _, z := range zonesResult.HostedZones {
		if aws.StringValue(z.Name) == zoneName {
			zoneID = z.Id
			break
		}
	}

	return zoneID, nil
}
