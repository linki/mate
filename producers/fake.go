package producers

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type endpointDefType int

const (
	none  endpointDefType = 0
	ipDef endpointDefType = 1 << iota
	hostnameDef
)

type fakeProducer struct {
	channel chan *pkg.Endpoint
}

func init() {
	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").Default("example.org.").StringVar(&params.dnsName)
	kingpin.Flag("fake-ip", "The fake IP addresse to use.").Default("8.8.8.8").StringVar(&params.ipAddress)

	rand.Seed(time.Now().UnixNano())
}

func NewFake() (*fakeProducer, error) {
	return &fakeProducer{
		channel: make(chan *pkg.Endpoint),
	}, nil
}

func (a *fakeProducer) Endpoints() ([]*pkg.Endpoint, error) {
	endpoints := make([]*pkg.Endpoint, 0)

	for i := 0; i < 10; i++ {
		endpoints = append(endpoints, genEndpoint())
	}

	return endpoints, nil
}

func (a *fakeProducer) StartWatch() error {
	for {
		a.channel <- genEndpoint()
		time.Sleep(5 * time.Second)
	}

	return nil
}

func (a *fakeProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}

func randomNumber(min, max int) int {
	return rand.Intn(max-min) + min
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz")

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func rollEndpointDef() endpointDefType {
	return endpointDefType(rand.Intn(int(ipDef|hostnameDef) + 1))
}

func (epd endpointDefType) hasIP() bool {
	return epd&ipDef != 0
}

func (epd endpointDefType) hasHostname() bool {
	return epd&hostnameDef != 0
}

func genHostname() string {
	return fmt.Sprintf("%s.%s", randomString(6), randomString(6))
}

func genEndpoint() *pkg.Endpoint {
	epd := rollEndpointDef()
	endpoint := &pkg.Endpoint{
		DNSName: fmt.Sprintf("%s.%s", randomString(2), params.dnsName),
	}

	if epd.hasIP() {
		endpoint.IP = net.IPv4(
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
		).String()
	}

	if epd.hasHostname() {
		endpoint.Hostname = genHostname()
	}

	return endpoint
}
