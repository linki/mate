package awsclient

import (
	"fmt"
	"testing"

	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	zoneID = "ABCDEFG"
	ttl    = int64(300)
)

func TestMapEndpointAlias(t *testing.T) {
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := mapEndpointAlias(ep, ttl, &zoneID)
	if *rsA.Type != "A" || *rsA.Name != pkg.SanitizeDNSName(ep.DNSName) ||
		*rsA.AliasTarget.DNSName != ep.Hostname ||
		*rsA.AliasTarget.HostedZoneId != zoneID {
		t.Error("Should create an A record")
	}
}

func TestMapEndpointTXT(t *testing.T) {
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsTXT := mapEndpointTXT(ep, ttl, "test")
	if *rsTXT.Type != "TXT" ||
		*rsTXT.Name != "example.com." ||
		len(rsTXT.ResourceRecords) != 1 ||
		*rsTXT.ResourceRecords[0].Value != "\"mate:test\"" {
		t.Error("Should create a TXT record")
	}
}

func TestFilterByGroupID(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	records := client.filterByGroupID([]*route53.ResourceRecordSet{
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("test.example.org."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("example-40123123.elb.amazon.com."),
				HostedZoneId: aws.String("myzone"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("test.example.org."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(fmt.Sprintf("\"mate:%s\"", groupID)),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("newtest.example.org."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("example-40124123.elb.amazon.com."),
				HostedZoneId: aws.String("myzone"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("newtest.example.org."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("\"mate:wrong\""), //wrong group id
				},
			},
		},
		&route53.ResourceRecordSet{ //without a TXT record
			Type: aws.String("A"),
			Name: aws.String("lonely1.example.org."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("example-40123123.elb.amazon.com."),
				HostedZoneId: aws.String("myzone"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("lonely2.example.org."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(fmt.Sprintf("\"mate:%s\"", groupID)),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("CNAME"),
			Name: aws.String("lonely2.example.org."),
		},
	})
	if len(records) != 3 || *records[0].Name != "test.example.org." ||
		*records[0].Type != "TXT" || *records[1].Name != "lonely2.example.org." ||
		*records[2].Type != "A" || *records[2].Name != "test.example.org." {
		t.Error("Should filter out only mate records with correct group ids")
	}
}

func TestGetTXTValue(t *testing.T) {
	if getTXTValue("test") != "\"mate:test\"" {
		t.Errorf("Should return TXT value of \"mate:test\", when test is passed")
	}
	if getTXTValue("") != "\"mate:\"" {
		t.Errorf("Should return TXT value of \"mate:\", when empty string is passed")
	}
}
