package awsclient

import (
	"fmt"

	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	EvaluateTargetHealth = true
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

//mapEndpointAlias ...
//create an AWS A Alias record
func mapEndpointAlias(ep *pkg.Endpoint, ttl int64, aliasHostedZoneID *string) *route53.ResourceRecordSet {
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

//mapEndpointTXT ...
//create an AWS TXT record
func mapEndpointTXT(ep *pkg.Endpoint, ttl int64, awsGroupID string) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(pkg.SanitizeDNSName(ep.DNSName)),
		TTL:  &ttl,
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(getTXTValue(awsGroupID)),
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

//filter out to include only mate created resource A and TXT records with the right group id
func (c *Client) filterByGroupID(rsets []*route53.ResourceRecordSet) []*route53.ResourceRecordSet {
	var hostnames []string
	var res []*route53.ResourceRecordSet
	for _, rs := range rsets {
		if aws.StringValue(rs.Type) == "TXT" && len(rs.ResourceRecords) == 1 {
			resource := rs.ResourceRecords[0]
			if aws.StringValue(resource.Value) == getTXTValue(c.options.GroupID) {
				hostnames = append(hostnames, *rs.Name)
				res = append(res, rs)
			}
		}
	}
	for _, rs := range rsets {
		if aws.StringValue(rs.Type) != "A" {
			continue
		}
		name := aws.StringValue(rs.Name)
		include := false
		for _, hostname := range hostnames {
			include = include || (pkg.SanitizeDNSName(name) == pkg.SanitizeDNSName(hostname))
		}
		if include {
			res = append(res, rs)
		}
	}
	return res
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

//return the Value of the TXT record as stored
func getTXTValue(groupID string) string {
	return fmt.Sprintf("\"mate:%s\"", groupID)
}
