package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
)

var (
	zoneID = "ABCDEFG"
)

func TestEndpointToAlias(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := client.endpointToAlias(ep, &zoneID)
	if *rsA.Type != "A" || *rsA.Name != pkg.SanitizeDNSName(ep.DNSName) ||
		*rsA.AliasTarget.DNSName != ep.Hostname ||
		*rsA.AliasTarget.HostedZoneId != zoneID {
		t.Error("Should create an A record")
	}
}

func TestGetAssignedTXTRecordObject(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := client.endpointToAlias(ep, &zoneID)
	rsTXT := client.GetAssignedTXTRecordObject(rsA)
	if *rsTXT.Type != "TXT" ||
		*rsTXT.Name != "example.com." ||
		len(rsTXT.ResourceRecords) != 1 ||
		*rsTXT.ResourceRecords[0].Value != "\"mate:test\"" {
		t.Error("Should create a TXT record")
	}
}

func TestGetGroupID(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	if client.GetGroupID() != "\"mate:test\"" {
		t.Errorf("Should return TXT value of \"mate:test\", when test is passed")
	}
}

func TestGetRecordTarget(t *testing.T) {
	r1 := &route53.ResourceRecordSet{
		Type: aws.String("A"),
		Name: aws.String("another.example.com."),
		AliasTarget: &route53.AliasTarget{
			DNSName:      aws.String("200.elb.com"),
			HostedZoneId: aws.String("123"),
		},
	}
	r2 := &route53.ResourceRecordSet{
		Type: aws.String("TXT"),
		Name: aws.String("another.example.com."),
		ResourceRecords: []*route53.ResourceRecord{
			&route53.ResourceRecord{
				Value: aws.String("ignored"),
			},
		},
	}
	r3 := &route53.ResourceRecordSet{
		Type: aws.String("CNAME"),
		Name: aws.String("cname.example.com."),
		ResourceRecords: []*route53.ResourceRecord{
			&route53.ResourceRecord{
				Value: aws.String("some-elb.amazon.com"),
			},
		},
	}

	if target := getRecordTarget(r1); target != "200.elb.com" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r1, "200.elb.com", target)
	}
	if target := getRecordTarget(r2); target != "" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r2, "", target)
	}
	if target := getRecordTarget(r3); target != "some-elb.amazon.com" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r3, "some-elb-amazon.com", target)
	}
}
