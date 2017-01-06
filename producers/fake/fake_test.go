package fake

import (
	"net"
	"net/url"
	"regexp"
	"testing"

	"github.com/zalando-incubator/mate/pkg"
)

func TestFakeReturnsTenEndpoints(t *testing.T) {
	endpoints := newEndpoints(t, nil)

	count := len(endpoints)
	if count != 10 {
		t.Error(count)
	}
}

func TestFakeEndpointsBelongToDomain(t *testing.T) {
	validRecord := regexp.MustCompile(`^.{2}\.example\.org\.$`)

	endpoints := newEndpoints(t, nil)

	for _, e := range endpoints {
		valid := validRecord.MatchString(e.DNSName)
		if !valid {
			t.Error(e.DNSName)
		}
	}
}

func TestFakeEndpointsResolveToIPAddresses(t *testing.T) {
	endpoints := newEndpoints(t, nil)

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

	endpoints := newEndpoints(t, producer)

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
	params.dnsName = "dnsName"
	params.mode = "mode"
	params.targetDomain = "targetDomain"

	producer, err := NewFake()
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

func newProducer() *fakeProducer {
	producer := &fakeProducer{
		mode:    ipMode,
		dnsName: "example.org.",
	}

	return producer
}

func newEndpoints(t *testing.T, producer *fakeProducer) []*pkg.Endpoint {
	if producer == nil {
		producer = newProducer()
	}

	endpoints, err := producer.Endpoints()
	if err != nil {
		t.Fatal(err.Error())
	}

	return endpoints
}
