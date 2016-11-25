package awsclient

import "testing"

func TestExtractELBName(t *testing.T) {
	elb0 := "aa-1231231-912313.eu-central-1.elb.amazonaws.com"
	elb1 := "aa1123-284152658.eu-central-1.elb.amazonaws.com"
	elb2 := "internal-aa1123-284152658.eu-central-1.elb.amazonaws.com"
	elb3 := "internal-aa-1123-284152658.eu-central-1.elb.amazonaws.com"
	res0 := extractELBName(elb0)
	res1 := extractELBName(elb1)
	res2 := extractELBName(elb2)
	res3 := extractELBName(elb3)
	if res0 != "aa-1231231" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb0, res0)
	}
	if res1 != "aa1123" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb1, res1)
	}
	if res2 != "aa1123" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb2, res2)
	}
	if res3 != "aa-1123" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb3, res3)
	}
}
