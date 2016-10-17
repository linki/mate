package consumers

import (
	"testing"

	"github.bus.zalan.do/teapot/mate/awsclient/awsclienttest"
	"github.bus.zalan.do/teapot/mate/pkg"
)

type awsTestItem struct {
	msg          string
	init         map[string]string
	sync         []*pkg.Endpoint
	process      *pkg.Endpoint
	fail         bool
	expectUpsert []*pkg.Endpoint
	expectDelete []*pkg.Endpoint
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

func checkEndpointSlices(got, expect []*pkg.Endpoint) bool {
	if len(got) != len(expect) {
		return false
	}

	for _, ep := range got {
		var found bool
		for _, eep := range expect {
			if ep.DNSName == eep.DNSName && ep.IP == eep.IP {
				found = true
				break
			}
		}

		if !found {
			return false
		}
	}

	return true
}

func testAWSConsumer(t *testing.T, ti awsTestItem) {
	client := &awsclienttest.Client{Records: ti.init}
	if ti.fail {
		client.FailNext()
	}

	if client.Records == nil {
		client.Records = make(map[string]string)
	}

	consumer := NewAWS(client)

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

	if !checkEndpointSlices(client.LastUpsert, ti.expectUpsert) {
		t.Error("failed to post the right upsert items", client.LastUpsert, ti.expectUpsert)
	}

	if !checkEndpointSlices(client.LastDelete, ti.expectDelete) {
		t.Error("failed to post the right delete items", client.LastDelete, ti.expectDelete)
	}
}

func TestAWSConsumer(t *testing.T) {
	for _, ti := range []awsTestItem{{
		msg: "no initial, no change",
	}, {
		msg: "no initial, sync new ones",
		sync: []*pkg.Endpoint{{
			"foo.org", "1.2.3.4",
		}, {
			"bar.org", "5.6.7.8",
		}},
		expectUpsert: []*pkg.Endpoint{{
			"foo.org", "1.2.3.4",
		}, {
			"bar.org", "5.6.7.8",
		}},
	}, {
		msg: "sync delete all",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		expectDelete: []*pkg.Endpoint{{
			"foo.org", "1.2.3.4",
		}, {
			"bar.org", "5.6.7.8",
		}},
	}, {
		msg: "insert, update, delete, leave",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
			"baz.org": "9.0.1.2",
		},
		sync: []*pkg.Endpoint{{
			"qux.org", "4.5.6.7",
		}, {
			"foo.org", "8.9.0.1",
		}, {
			"baz.org", "9.0.1.2",
		}},
		expectUpsert: []*pkg.Endpoint{{
			"qux.org", "4.5.6.7",
		}, {
			"foo.org", "8.9.0.1",
		}},
		expectDelete: []*pkg.Endpoint{{
			"bar.org", "5.6.7.8",
		}},
	}, {
		msg: "fail on list",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		sync: []*pkg.Endpoint{{
			"baz.org", "9.0.1.2",
		}},
		fail:       true,
		expectFail: true,
	}, {
		msg: "fail on change",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		sync: []*pkg.Endpoint{{
			"baz.org", "9.0.1.2",
		}},
		fail:       true,
		expectFail: true,
	}, {
		msg: "process existing",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		process:      &pkg.Endpoint{DNSName: "foo.org", IP: "2.3.4.5"},
		expectUpsert: []*pkg.Endpoint{{DNSName: "foo.org", IP: "2.3.4.5"}},
	}, {
		msg: "process new",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		process:      &pkg.Endpoint{DNSName: "baz.org", IP: "9.0.1.2"},
		expectUpsert: []*pkg.Endpoint{{DNSName: "baz.org", IP: "9.0.1.2"}},
	}, {
		msg: "fail on process",
		init: map[string]string{
			"foo.org": "1.2.3.4",
			"bar.org": "5.6.7.8",
		},
		process:    &pkg.Endpoint{DNSName: "foo.org", IP: "2.3.4.5"},
		fail:       true,
		expectFail: true,
	}} {
		t.Run(ti.msg, func(t *testing.T) {
			testAWSConsumer(t, ti)
		})
	}
}
