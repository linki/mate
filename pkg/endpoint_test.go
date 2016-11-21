package pkg

import "testing"

func TestSanitizeDNSName(t *testing.T) {
	dns1 := ""
	res1 := SanitizeDNSName(dns1)
	if res1 != "." {
		t.Errorf("SanitizeDNSName failed for %s", dns1)
	}
	dns2 := "example.com"
	res2 := SanitizeDNSName(dns2)
	if res2 != "example.com." {
		t.Errorf("SanitizeDNSName failed for %s", dns2)
	}
	dns3 := "example.com."
	res3 := SanitizeDNSName(dns3)
	if res3 != "example.com." {
		t.Errorf("SanitizeDNSName failed for %s", dns3)
	}
}
