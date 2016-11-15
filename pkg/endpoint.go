package pkg

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/aws"
	"strings"
	"fmt"
)

var (
	EvaluateTargetHealth = true
	clusterName = "dummy" //this needs to be replaced by quering for cluster name in kubernetes
)

// Endpoint is used to pass data from the producer to the consumer.
type Endpoint struct {

	// The DNS name to be set by the consumer for the record.
	DNSName string

	// In case of A records, the IP address value of the record.
	IP string

	// The value of ALIAS record (preferrably) or the CNAME
	// record, in case the provider receives only a hostname for
	// the service.
	Hostname string
}

//FQDN ...
//return the fully qualified domain name with trailing dot
func FQDN(dns string) string{
	return strings.Trim(dns, ".") + ".";
}

//AWSARecordAlias ...
//create an AWS A Alias record
func (ep *Endpoint) AWSARecordAlias(zoneID *string, ttl int64) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(ep.DNSName),
		TTL: &ttl,
		AliasTarget: &route53.AliasTarget{
			DNSName: &ep.Hostname,
			EvaluateTargetHealth: &EvaluateTargetHealth,
			HostedZoneId: zoneID,
		},
	}
	return rs
}

//AWSTXTRecord ...
//create a AWS TXT record
func (ep *Endpoint) AWSTXTRecord(ttl int64) *route53.ResourceRecordSet{
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(ep.DNSName),
		TTL: &ttl,
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(fmt.Sprintf("\"mate:%s\"", clusterName)),
		}},
	}
	return rs
}

