package pkg

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	zoneID = "ABCDEFG"
	ttl    = int64(300)
)

func Test_AWSARecordAlias(t *testing.T) {
	ep := &Endpoint{
		DNSName:     "example.com",
		IP:          "10.202.10.123",
		Hostname:    "amazon.elb.com",
		AliasZoneID: "00ER123",
	}
	rsA := ep.AWSARecordAlias(ttl)
	assert.Equal(t, *rsA.Type, "A", "Create an A Record")
}

func Test_AWSTXTRecord(t *testing.T) {
	ep := &Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsTXT := ep.AWSTXTRecord(ttl, "test")
	assert.Equal(t, *rsTXT.Type, "TXT", "Create TXT Record")
	assert.Equal(t, *rsTXT.Name, "example.com.", "Name should match")
	assert.Equal(t, len(rsTXT.ResourceRecords), 1, "Should include one resource record")
	assert.Equal(t, *rsTXT.ResourceRecords[0].Value, "\"mate:test\"")
}

func TestFQDN(t *testing.T) {
	dns1 := ""
	res1 := FQDN(dns1)
	if res1 != "." {
		t.Errorf("FQDN failed for %s", dns1)
	}
	dns2 := "example.com"
	res2 := FQDN(dns2)
	if res2 != "example.com." {
		t.Errorf("FQDN failed for %s", dns2)
	}
	dns3 := "example.com."
	res3 := FQDN(dns3)
	if res3 != "example.com." {
		t.Errorf("FQDN failed for %s", dns3)
	}
}
