package awsclient

import (
	"testing"

	"github.bus.zalan.do/teapot/mate/pkg"
)

var (
	zoneID = "ABCDEFG"
)

func TestMapEndpointAlias(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := client.MapEndpointAlias(ep, &zoneID)
	if *rsA.Type != "A" || *rsA.Name != pkg.SanitizeDNSName(ep.DNSName) ||
		*rsA.AliasTarget.DNSName != ep.Hostname ||
		*rsA.AliasTarget.HostedZoneId != zoneID {
		t.Error("Should create an A record")
	}
}

func TestMapEndpointTXT(t *testing.T) {
	groupID := "test"
	client := New(Options{GroupID: groupID})
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsTXT := client.MapEndpointTXT(ep)
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
