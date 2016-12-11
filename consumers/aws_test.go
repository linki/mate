package consumers

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/zalando-incubator/mate/pkg"
	awsclient "github.com/zalando-incubator/mate/pkg/aws"
	awstest "github.com/zalando-incubator/mate/pkg/aws/test"
)

type awsTestItem struct {
	msg        string
	init       []*route53.ResourceRecordSet
	sync       []*pkg.Endpoint
	process    *pkg.Endpoint
	fail       bool
	expect     map[string]awstest.Result
	expectFail bool
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
				if *eep.Type == "A" && *eep.AliasTarget.DNSName == *ep.AliasTarget.DNSName && *ep.Name == *eep.Name {
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
	client := &awstest.Client{
		Client: awsclient.New(awsclient.Options{
			GroupID: groupID,
		}),
		Zones: map[string]string{
			"example.com.": "example.com.",
			"new.com.":     "new.com.",
		},
		Options: awstest.Options{
			RecordSetTTL: 10,
			GroupID:      groupID,
		}}
	if ti.fail {
		client.FailNext()
	}
	newCom := []*route53.ResourceRecordSet{
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("new.com."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("200.elb.com"),
				HostedZoneId: aws.String("123"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("new.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.Client.GetGroupID()),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("de.new.com."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("200.elb.com"),
				HostedZoneId: aws.String("123"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("de.new.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("new-id"),
				},
			},
		},
	}
	exampleCom := []*route53.ResourceRecordSet{
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
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.Client.GetGroupID()),
				},
			},
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
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.Client.GetGroupID()),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("another.example.com."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("200.elb.com"),
				HostedZoneId: aws.String("123"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("another.example.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("ignored"),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("CNAME"),
			Name: aws.String("cname.example.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("some-elb.amazon.com"),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("withoutA.example.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("lonely"),
				},
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("withouttxt.example.com."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("random.elb.com"),
				HostedZoneId: aws.String("123"),
			},
		},
	}
	client.Current = map[string][]*route53.ResourceRecordSet{
		"example.com.": exampleCom,
		"new.com.":     newCom,
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

	for zone := range client.Zones {
		if !checkEndpointSlices(client.Output[zone].LastUpsert, ti.expect[zone].LastUpsert) {
			t.Error("failed to post the right upsert items", client.Output[zone].LastUpsert, ti.expect[zone].LastUpsert)
		}
		if !checkEndpointSlices(client.Output[zone].LastCreate, ti.expect[zone].LastCreate) {
			t.Error("failed to post the right create items", client.Output[zone].LastCreate, ti.expect[zone].LastCreate)
		}
		if !checkEndpointSlices(client.Output[zone].LastDelete, ti.expect[zone].LastDelete) {
			t.Error("failed to post the right delete items", client.Output[zone].LastDelete, ti.expect[zone].LastDelete)
		}
	}
}

func TestAWSConsumer(t *testing.T) { //exclude IP endpoints for now only Alias works
	for _, ti := range []awsTestItem{{
		msg: "no initial, sync new ones",
		sync: []*pkg.Endpoint{{
			"test.example.com", "", "abc.def.ghi",
		}, {
			"withouttxt.example.com", "", "random.com",
		}},
		expect: map[string]awstest.Result{
			"example.com.": awstest.Result{
				LastUpsert: []*route53.ResourceRecordSet{
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
				LastDelete: []*route53.ResourceRecordSet{
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
			"new.com.": awstest.Result{
				LastDelete: []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("new.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("200.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("new.com."),
					},
				},
			},
		},
	}, {
		msg: "sync delete all",
		sync: []*pkg.Endpoint{{
			"another.example.com", "", "abc.def.ghi",
		}, {
			"cname.example.com", "", "hello.elb.com",
		}, {
			"new.com", "", "elb.com",
		}},
		expect: map[string]awstest.Result{
			"example.com.": awstest.Result{
				LastDelete: []*route53.ResourceRecordSet{
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
			"new.com.": awstest.Result{
				LastUpsert: []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("new.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("new.com."),
					},
				},
			},
		},
	}, {
		msg: "insert, update, delete, leave",
		sync: []*pkg.Endpoint{{
			"new.example.com", "", "qux.elb",
		}, {
			"test.example.com", "", "foo.elb2",
		}},
		expect: map[string]awstest.Result{
			"example.com.": awstest.Result{
				LastUpsert: []*route53.ResourceRecordSet{
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
				LastCreate: []*route53.ResourceRecordSet{},
				LastDelete: []*route53.ResourceRecordSet{
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
			"new.com.": awstest.Result{
				LastDelete: []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("new.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("200.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("new.com."),
					},
				},
			},
		},
	}, {
		msg:     "process wrong hostname",
		process: &pkg.Endpoint{DNSName: "baz.org", Hostname: "cool.elb"},
		expect: map[string]awstest.Result{
			"example.com.": awstest.Result{
				LastCreate: []*route53.ResourceRecordSet{},
			},
		},
	}, {
		msg:     "process correct hostname",
		process: &pkg.Endpoint{DNSName: "example.com", Hostname: "cool.elb"},
		expect: map[string]awstest.Result{
			"example.com.": awstest.Result{
				LastCreate: []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("example.com."),
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
	},
	} {
		t.Run(ti.msg, func(t *testing.T) {
			testAWSConsumer(t, ti)
		})
	}
}
