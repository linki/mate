package consumers

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awstest "github.com/zalando-incubator/mate/pkg/aws/test"
)

type awsTestItem struct {
	msg          string
	sync         []*pkg.Endpoint
	process      *pkg.Endpoint
	fail         bool
	expectCreate map[string][]*route53.ResourceRecordSet
	expectUpsert map[string][]*route53.ResourceRecordSet
	expectDelete map[string][]*route53.ResourceRecordSet
	expectFail   bool
}

func checkTestError(t *testing.T, err error, expect bool) bool {
	if err == nil && expect {
		t.Error("failed to fail")
		return false
	}

	if err != nil && !expect {
		t.Error("unexpected error", err)
		return false
	}

	return true
}

func checkEndpointSlices(got []*route53.ResourceRecordSet, expect []*route53.ResourceRecordSet) bool {
	if len(got) != len(expect) {
		return false
	}
	for _, ep := range got {
		if *ep.Type != "A" {
			continue
		}
		var found bool
		for _, eep := range expect {
			if *ep.Type == "A" {
				if *eep.Type == "A" && pkg.SanitizeDNSName(*eep.AliasTarget.DNSName) == pkg.SanitizeDNSName(*ep.AliasTarget.DNSName) &&
					*ep.Name == *eep.Name {
					found = true
				}
				continue
			}
			if *ep.Type == "TXT" {
				if *eep.Type == "TXT" && *ep.Name == *eep.Name {
					found = true
				}
				continue
			}
			return false
		}
		if !found {
			return false
		}
	}

	return true
}

func testAWSConsumer(t *testing.T, ti awsTestItem) {
	groupID := "testing-group-id"
	client := awstest.NewClient(groupID)
	if ti.fail {
		client.FailNext()
	}

	consumer := withClient(client)

	if ti.process == nil {
		err := consumer.Sync(ti.sync)
		if !checkTestError(t, err, ti.expectFail) {
			return
		}
	} else {
		err := consumer.Process(ti.process)
		if !checkTestError(t, err, ti.expectFail) {
			return
		}
	}
	if len(client.LastUpsert) != len(ti.expectUpsert) {
		t.Error("failed to post the right upsert items. Number of hosted zones is different.", client.LastUpsert, ti.expectUpsert)
	}
	for zoneName := range ti.expectUpsert {
		if !checkEndpointSlices(client.LastUpsert[zoneName], ti.expectUpsert[zoneName]) {
			t.Error("failed to post the right upsert items", client.LastUpsert[zoneName], ti.expectUpsert[zoneName])
		}
	}
	if len(client.LastCreate) != len(ti.expectCreate) {
		t.Error("failed to post the right upsert items. Number of hosted zones is different.", client.LastUpsert, ti.expectUpsert)
	}
	for zoneName := range ti.expectCreate {
		if !checkEndpointSlices(client.LastCreate[zoneName], ti.expectCreate[zoneName]) {
			t.Error("failed to post the right create items", client.LastCreate[zoneName], ti.expectCreate[zoneName])
		}
	}
	if len(client.LastDelete) != len(ti.expectDelete) {
		t.Error("failed to post the right upsert items. Number of hosted zones is different.", client.LastUpsert, ti.expectUpsert)
	}
	for zoneName := range ti.expectDelete {
		if !checkEndpointSlices(client.LastDelete[zoneName], ti.expectDelete[zoneName]) {
			t.Error("failed to post the right delete items", client.LastDelete[zoneName], ti.expectDelete[zoneName])
		}
	}
}

func TestAWSConsumer(t *testing.T) { //exclude IP endpoints for now only Alias works
	for _, ti := range []awsTestItem{
		{
			msg: "partial overlap",
			sync: []*pkg.Endpoint{
				{
					"test.example.com", "", "404.elb.com",
				},
				{
					"update.example.com", "", "elb.com",
				},
				{
					"withouttxt.example.com", "", "random.com",
				},
			},
			expectUpsert: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.example.com."),
					},
				},
			},
		},
		{
			msg: "no initial, sync new ones",
			sync: []*pkg.Endpoint{{
				"test.example.com", "", "abc.def.ghi",
			}, {
				"withouttxt.example.com", "", "random.com",
			}},
			expectUpsert: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("test.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("abc.def.ghi"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("test.example.com."),
					},
				},
			},
			expectDelete: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("302.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.example.com."),
					},
				},
			},
		},
		{
			msg: "sync delete all",
			sync: []*pkg.Endpoint{{
				"another.example.com", "", "abc.def.ghi",
			}, {
				"cname.example.com", "", "hello.elb.com",
			}},
			expectDelete: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("test.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("404.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("test.example.com."),
					},
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("302.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.example.com."),
					},
				},
			},
		}, {
			msg: "insert, update, delete, leave",
			sync: []*pkg.Endpoint{{
				"new.example.com", "", "qux.elb",
			}, {
				"test.example.com", "", "foo.elb2",
			}, {
				"test.foo.com", "", "foo.loadbalancer", //skip it
			}, {
				"update.foo.com", "", "new.loadbalancer",
			}},
			expectUpsert: map[string][]*route53.ResourceRecordSet{
				"foo.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.foo.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("new.loadbalancer"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.foo.com."),
					},
				},
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("test.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("foo.elb2"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("test.example.com."),
					},
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("new.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("qux.elb"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("new.example.com."),
					},
				},
			},
			expectDelete: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("302.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.example.com."),
					},
				},
			},
		}, {
			msg:     "process new",
			process: &pkg.Endpoint{DNSName: "process.example.com.", Hostname: "cool.elb"},
			expectCreate: map[string][]*route53.ResourceRecordSet{
				"example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("process.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("cool.elb"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("baz.org."),
					},
				},
			},
		},
	} {
		t.Run(ti.msg, func(t *testing.T) {
			testAWSConsumer(t, ti)
		})
	}
}
