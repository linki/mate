package awsclient

import "testing"

func TestExtractELBName(t *testing.T) {
	elb1 := "aa1123-284152658.eu-central-1.elb.amazonaws.com"
	elb2 := "aa-1123-284152658.eu-central-1.elb.amazonaws.com"
	res1 := extractELBName(elb1)
	res2 := extractELBName(elb2)
	if res1 != "aa1123" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb1, res1)
	}
	if res2 != "aa-1123" {
		t.Errorf("Incorrect ELB name for %s, got %s", elb2, res2)
	}
}
