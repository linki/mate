package aws

import (
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

//endpointToAlias convert endpoint to an AWS A Alias record
func (c *Client) endpointToAlias(ep *pkg.Endpoint, canonicalZoneID *string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              aws.String(pkg.SanitizeDNSName(ep.Hostname)),
			EvaluateTargetHealth: aws.Bool(evaluateTargetHealth),
			HostedZoneId:         canonicalZoneID,
		},
	}
	return rs
}

func getRecordTarget(r *route53.ResourceRecordSet) string {
	if aws.StringValue(r.Type) == "TXT" {
		return ""
	}
	if r.AliasTarget != nil {
		return aws.StringValue(r.AliasTarget.DNSName)
	}
	return aws.StringValue(r.ResourceRecords[0].Value)
}

//createTXTRecordObject creates AWS TXT Record Resource object for a given DNS entry
func (c *Client) createTXTRecordObject(DNSName string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(pkg.SanitizeDNSName(DNSName)),
		TTL:  aws.Int64(defaultTxtTTL),
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(c.GetGroupID()),
		}},
	}
	return rs
}

func createChangesList(action string, rsets []*route53.ResourceRecordSet) []*route53.Change {
	var changes []*route53.Change
	for _, rset := range rsets {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: rset,
		})
	}
	return changes
}
