package producers

import (
	"net"
	"regexp"
	"testing"

	"github.bus.zalan.do/teapot/mate/pkg"
)

func TestFakeReturnsTenEndpoints(t *testing.T) {
	endpoints := newEndpoints(t)

	count := len(endpoints)
	if count != 10 {
		t.Error(count)
	}
}

func TestFakeEndpointsBelongToDomain(t *testing.T) {
	validRecord := regexp.MustCompile(`^.{2}\.example\.org\.$`)

	endpoints := newEndpoints(t)

	for _, e := range endpoints {
		valid := validRecord.MatchString(e.DNSName)
		if !valid {
			t.Error(e.DNSName)
		}
	}
}

func TestFakeEndpointsResolveToIPAddresses(t *testing.T) {
	endpoints := newEndpoints(t)

	for _, e := range endpoints {
		ip := net.ParseIP(e.IP)
		if ip == nil {
			t.Error(ip)
		}
	}
}

func newProducer() *fakeProducer {
	producer := &fakeProducer{
		channel: make(chan *pkg.Endpoint),
		mode:    ipMode,
		dnsName: "example.org.",
	}

	return producer
}

func newEndpoints(t *testing.T) []*pkg.Endpoint {
	producer := newProducer()

	endpoints, err := producer.Endpoints()
	if err != nil {
		t.Error(err.Error())
	}

	return endpoints
}
