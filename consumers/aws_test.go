package consumers

import (
	"fmt"
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

func TestEndpointToAlias(t *testing.T) {
	groupID := "test"
	zoneID := "test"
	client := &awsConsumer{
		groupID: groupID,
	}
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := client.endpointToAlias(ep, &zoneID)
	if *rsA.Type != "A" || *rsA.Name != pkg.SanitizeDNSName(ep.DNSName) ||
		*rsA.AliasTarget.DNSName != pkg.SanitizeDNSName(ep.Hostname) ||
		*rsA.AliasTarget.HostedZoneId != zoneID {
		t.Error("Should create an A record")
	}
}

func TestGetAssignedTXTRecordObject(t *testing.T) {
	groupID := "test"
	zoneID := "test"
	client := &awsConsumer{
		groupID: groupID,
	}
	ep := &pkg.Endpoint{
		DNSName:  "example.com",
		IP:       "10.202.10.123",
		Hostname: "amazon.elb.com",
	}
	rsA := client.endpointToAlias(ep, &zoneID)
	rsTXT := client.getAssignedTXTRecordObject(rsA)
	if *rsTXT.Type != "TXT" ||
		*rsTXT.Name != "example.com." ||
		len(rsTXT.ResourceRecords) != 1 ||
		*rsTXT.ResourceRecords[0].Value != "\"mate:test\"" {
		t.Error("Should create a TXT record")
	}
}

func TestGetZoneIDForEndpoint(t *testing.T) {
	hostedZonesMap := map[string]string{
		"example.com":                    "id1",
		"test.com":                       "id2",
		"sub.test.com":                   "id3",
		"long-sub1.internal.example.com": "id4",
		"long-sub2.internal.example.com": "id5",
	}
	record1 := &route53.ResourceRecordSet{
		Name: aws.String("name.example.com"),
	}
	record2 := &route53.ResourceRecordSet{
		Name: aws.String("name.example.test.com"),
	}
	record3 := &route53.ResourceRecordSet{
		Name: aws.String("name.sub.test.com"),
	}
	record4 := &route53.ResourceRecordSet{
		Name: aws.String("name.long-sub1.internal.example.com"),
	}
	record5 := &route53.ResourceRecordSet{
		Name: aws.String("name.long-sub2.internal.example.com"),
	}
	if getZoneIDForEndpoint(hostedZonesMap, record1) != "id1" {
		t.Errorf("Incorrect zone id for %v", record1)
	}
	if getZoneIDForEndpoint(hostedZonesMap, record2) != "id2" {
		t.Errorf("Incorrect zone id for %v", record2)
	}
	if getZoneIDForEndpoint(hostedZonesMap, record3) != "id3" {
		t.Errorf("Incorrect zone id for %v", record3)
	}
	if getZoneIDForEndpoint(hostedZonesMap, record4) != "id4" {
		t.Errorf("Incorrect zone id for %v", record4)
	}
	if getZoneIDForEndpoint(hostedZonesMap, record5) != "id5" {
		t.Errorf("Incorrect zone id for %v", record5)
	}
}

func sameTargets(lb1, lb2 string) bool {
	return lb1 == lb2
}

func TestRecordInfo(t *testing.T) {
	groupID := "test"
	client := &awsConsumer{
		groupID: groupID,
	}
	records := []*route53.ResourceRecordSet{
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
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.getGroupID()),
				},
			},
		},
	}
	recordInfoMap := client.recordInfo(records)
	if len(recordInfoMap) != 1 {
		t.Errorf("Incorrect record info for %v", records)
	}
	if val, exist := recordInfoMap["test.example.com."]; !exist {
		t.Errorf("Incorrect record info for %v", records)
	} else {
		if val.GroupID != client.getGroupID() {
			t.Errorf("Incorrect record info for %v", records)
		}
		if !sameTargets("abc.def.ghi", val.Target) {
			t.Errorf("Incorrect record info for %v", records)
		}
	}
	records = []*route53.ResourceRecordSet{
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("test.example.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.getGroupID()),
				},
			},
		},
	}
	recordInfoMap = client.recordInfo(records)
	if len(recordInfoMap) != 1 {
		t.Errorf("Incorrect record info for %v", records)
	}
	if val, exist := recordInfoMap["test.example.com."]; !exist {
		t.Errorf("Incorrect record info for %v", records)
	} else {
		if val.GroupID != client.getGroupID() {
			t.Errorf("Incorrect record info for %v", records)
		}
		if !sameTargets("", val.Target) {
			t.Errorf("Incorrect record info for %v", records)
		}
	}

	records = []*route53.ResourceRecordSet{
		&route53.ResourceRecordSet{
			Type: aws.String("A"),
			Name: aws.String("new.example.com."),
			AliasTarget: &route53.AliasTarget{
				DNSName:      aws.String("elb.com"),
				HostedZoneId: aws.String("123"),
			},
		},
		&route53.ResourceRecordSet{
			Type: aws.String("TXT"),
			Name: aws.String("new.example.com."),
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String("mate:new-group-id"),
				},
			},
		},
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
			ResourceRecords: []*route53.ResourceRecord{
				&route53.ResourceRecord{
					Value: aws.String(client.getGroupID()),
				},
			},
		},
	}
	recordInfoMap = client.recordInfo(records)
	if len(recordInfoMap) != 2 {
		t.Errorf("Incorrect record info for %v", records)
	}
	if val, exist := recordInfoMap["test.example.com."]; !exist {
		t.Errorf("Incorrect record info for %v", records)
	} else {
		if val.GroupID != client.getGroupID() {
			t.Errorf("Incorrect record info for %v", records)
		}
		if !sameTargets("abc.def.ghi", val.Target) {
			t.Errorf("Incorrect record info for %v", records)
		}
	}
	if val, exist := recordInfoMap["new.example.com."]; !exist {
		t.Errorf("Incorrect record info for %v", records)
	} else {
		if val.GroupID != "mate:new-group-id" {
			t.Errorf("Incorrect record info for %v", records)
		}
		if !sameTargets("elb.com", val.Target) {
			t.Errorf("Incorrect record info for %v", records)
		}
	}
}

func TestGetGroupID(t *testing.T) {
	groupID := "test"
	client := &awsConsumer{
		groupID: groupID,
	}
	if client.getGroupID() != "\"mate:test\"" {
		t.Errorf("Should return TXT value of \"mate:test\", when test is passed")
	}
}

func TestGetRecordTarget(t *testing.T) {
	groupID := "test"
	client := &awsConsumer{
		groupID: groupID,
	}
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

	if target := client.getRecordTarget(r1); target != "200.elb.com" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r1, "200.elb.com", target)
	}
	if target := client.getRecordTarget(r2); target != "" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r2, "", target)
	}
	if target := client.getRecordTarget(r3); target != "some-elb.amazon.com" {
		t.Errorf("Incorrect target extracted for %v, expected: %s, got: %s", r3, "some-elb-amazon.com", target)
	}
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

func NonEmptyMapLength(Map map[string][]*route53.ResourceRecordSet) int {
	ans := 0
	for key := range Map {
		if len(Map[key]) > 0 {
			ans++
		}
	}
	return ans
}

func testAWSConsumer(t *testing.T, ti awsTestItem) {
	groupID := "testing-group-id"
	client := awstest.NewClient(groupID, awstest.GetOriginalState(fmt.Sprintf("\"mate:%s\"", groupID)), awstest.GetHostedZones())

	consumer := withClient(client, groupID)

	if ti.process == nil {
		consumer.Sync(ti.sync)
	} else {
		consumer.Process(ti.process)
	}
	if NonEmptyMapLength(client.LastUpsert) != NonEmptyMapLength(ti.expectUpsert) {
		t.Error("failed to post the right upsert items. Number of hosted zones is different.", client.LastUpsert, ti.expectUpsert)
	}
	for zoneName := range ti.expectUpsert {
		if !checkEndpointSlices(client.LastUpsert[zoneName], ti.expectUpsert[zoneName]) {
			t.Error("failed to post the right upsert items", client.LastUpsert[zoneName], ti.expectUpsert[zoneName])
		}
	}
	if NonEmptyMapLength(client.LastCreate) != NonEmptyMapLength(ti.expectCreate) {
		t.Error("failed to post the right create items. Number of hosted zones is different.", client.LastCreate, ti.expectCreate)
	}
	for zoneName := range ti.expectCreate {
		if !checkEndpointSlices(client.LastCreate[zoneName], ti.expectCreate[zoneName]) {
			t.Error("failed to post the right create items", client.LastCreate[zoneName], ti.expectCreate[zoneName])
		}
	}
	if NonEmptyMapLength(client.LastDelete) != NonEmptyMapLength(ti.expectDelete) {
		t.Error("failed to post the right delete items. Number of hosted zones is different.", client.LastDelete, ti.expectDelete)
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
				{
					"nest.sub.example.com", "", "nested.elb",
				},
			},
			expectUpsert: map[string][]*route53.ResourceRecordSet{
				"sub.example.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("nest.sub.example.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("nested.elb"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("nest.sub.example.com."),
					},
				},
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
			expectDelete: map[string][]*route53.ResourceRecordSet{
				"foo.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.foo.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("404.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.foo.com."),
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
				"foo.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.foo.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("404.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.foo.com."),
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
				"foo.com.": []*route53.ResourceRecordSet{
					&route53.ResourceRecordSet{
						Type: aws.String("A"),
						Name: aws.String("update.foo.com."),
						AliasTarget: &route53.AliasTarget{
							DNSName:      aws.String("404.elb.com"),
							HostedZoneId: aws.String("123"),
						},
					},
					&route53.ResourceRecordSet{
						Type: aws.String("TXT"),
						Name: aws.String("update.foo.com."),
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
