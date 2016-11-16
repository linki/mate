package pkg

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	EvaluateTargetHealth = true
	clusterName          = "dummy" //this needs to be replaced by quering for cluster name in kubernetes
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

	// Alias Zone Id for the Alias A records in AWS
	AliasZoneID string
}

//FQDN ...
//return the fully qualified domain name with trailing dot
func FQDN(dns string) string {
	return strings.Trim(dns, ".") + "."
}

//AWSARecordAlias ...
//create an AWS A Alias record
func (ep *Endpoint) AWSARecordAlias(ttl int64) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String(FQDN(ep.DNSName)),
		AliasTarget: &route53.AliasTarget{
			DNSName:              &ep.Hostname,
			EvaluateTargetHealth: &EvaluateTargetHealth,
			HostedZoneId:         &ep.AliasZoneID,
		},
	}
	return rs
}

//AWSTXTRecord ...
//create a AWS TXT record
func (ep *Endpoint) AWSTXTRecord(ttl int64) *route53.ResourceRecordSet {
	rs := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String(FQDN(ep.DNSName)),
		TTL:  &ttl,
		ResourceRecords: []*route53.ResourceRecord{{
			Value: aws.String(GetMateValue(clusterName)),
		}},
	}
	return rs
}

//GetMateValue ...
//convert to mate value in a TXT record
func GetMateValue(clusterName string) string {
	return fmt.Sprintf("\"mate:%s\"", clusterName)
}
