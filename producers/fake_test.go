package producers

import (
	"net"
	"net/url"
	"regexp"
	"testing"

	"github.com/zalando-incubator/mate/pkg"
)

func TestFakeReturnsTenEndpoints(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)

	count := len(endpoints)
	if count != 10 {
		t.Error(count)
	}
}

func TestFakeEndpointsBelongToDomain(t *testing.T) {
	validRecord := regexp.MustCompile(`^.{2}\.example\.org\.$`)

	endpoints := newFakeEndpoints(t, nil)

	for _, e := range endpoints {
		valid := validRecord.MatchString(e.DNSName)
		if !valid {
			t.Error(e.DNSName)
		}
	}
}

func TestFakeEndpointsResolveToIPAddresses(t *testing.T) {
	endpoints := newFakeEndpoints(t, nil)

	for _, e := range endpoints {
		ip := net.ParseIP(e.IP)
		if ip == nil {
			t.Error(ip)
		}
	}
}

func TestFakeEndpointsResolveToHostnamesInHostnameMode(t *testing.T) {
	producer := &fakeProducer{
		mode:    hostnameMode,
		dnsName: "example.org.",
	}

	endpoints := newFakeEndpoints(t, producer)

	for _, e := range endpoints {
		if e.Hostname == "" {
			t.Error("missing hostname")
		}

		_, err := url.Parse(e.Hostname)
		if err != nil {
			t.Error(err)
		}
	}
}

func TestNewFakeReadsConfigurationFromParams(t *testing.T) {
	fakeParams := &FakeProducerOptions{}
	fakeParams.DNSName = "dnsName"
	fakeParams.Mode = "mode"
	fakeParams.TargetDomain = "targetDomain"

	producer, err := NewFake(fakeParams)
	if err != nil {
		t.Fatal(err)
	}

	if producer.dnsName != "dnsName" {
		t.Error(producer.dnsName)
	}

	if producer.mode != "mode" {
		t.Error(producer.mode)
	}

	if producer.targetDomain != "targetDomain" {
		t.Error(producer.targetDomain)
	}
}

func newFakeProducer() *fakeProducer {
	producer := &fakeProducer{
		mode:    ipMode,
		dnsName: "example.org.",
	}

	return producer
}

func newFakeEndpoints(t *testing.T, producer *fakeProducer) []*pkg.Endpoint {
	if producer == nil {
		producer = newFakeProducer()
	}

	endpoints, err := producer.Endpoints()
	if err != nil {
		t.Fatal(err.Error())
	}

	return endpoints
}
