package awsclient

import "testing"
import "github.com/stretchr/testify/assert"

func TestExtractELBName(t *testing.T) {
	elb1 := "aa1123-284152658.eu-central-1.elb.amazonaws.com"
	elb2 := "aa-1123-284152658.eu-central-1.elb.amazonaws.com"

	assert.Equal(t, extractELBName(elb1), "aa1123", "No hyphen ELB name extraction")
	assert.Equal(t, extractELBName(elb2), "aa-1123", "Hyphened ELB name extraction")
}
