package awsclient

import (
	"os"
	"strings"
	"testing"

	"github.bus.zalan.do/teapot/mate/pkg"
)

const (
	awsProviderVarName   = "AWS_PROVIDER_TEST"
	awsHostedZoneVarName = "AWS_HOSTED_ZONE"
	awsAccountIDVarName  = "AWS_ACCOUNT_ID"
	awsRoleVarName       = "AWS_ROLE"
)

func logSets(t *testing.T, sets []*pkg.Endpoint) {
	for _, s := range sets {
		t.Log(s.DNSName, s.IP)
	}
}

func TestAWSWithProvider(t *testing.T) {
	if os.Getenv(awsProviderVarName) != "true" {
		t.Skip()
	}

	zone := os.Getenv(awsHostedZoneVarName)
	client := New(Options{
		AccountID:  os.Getenv(awsAccountIDVarName),
		Role:       os.Getenv(awsRoleVarName),
		HostedZone: zone,
	})

	sets, err := client.ListRecordSets()
	if err != nil {
		t.Error(err)
		return
	}

	t.Log("found initially:")
	logSets(t, sets)

	t.Log("deleting all")
	if err := client.ChangeRecordSets(nil, sets); err != nil {
		t.Error(err)
		return
	}

	sets, err = client.ListRecordSets()
	if err != nil {
		t.Error(err)
		return
	}

	if len(sets) != 0 {
		t.Error("failed to delete sets")
		return
	}

	sets = []*pkg.Endpoint{{
		DNSName: "test1.mate.teapot.zalan.do", IP: "1.2.3.4",
	}, {
		DNSName: "test2.mate.teapot.zalan.do", IP: "5.6.7.8",
	}}

	t.Log("creating record sets:")
	logSets(t, sets)
	if err := client.ChangeRecordSets(sets, nil); err != nil {
		t.Error(err)
		return
	}

	setsCheck, err := client.ListRecordSets()
	if err != nil {
		t.Error(err)
		return
	}

	const checkMessage = "failed to return all the record sets"

	if len(setsCheck) != len(sets) {
		t.Error(checkMessage)
		return
	}

	t.Log("got sets back:")
	logSets(t, setsCheck)
	for _, sc := range setsCheck {
		var found bool
		for _, s := range sets {
			if strings.TrimSuffix(s.DNSName, ".") == strings.TrimSuffix(sc.DNSName, ".") &&
				s.IP == sc.IP {
				found = true
				break
			}
		}

		if !found {
			t.Error(checkMessage)
			return
		}
	}
}
