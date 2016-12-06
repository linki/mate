package fake

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	defaultDomain = "example.org."
	ipMode        = "ip"
	hostnameMode  = "hostname"
)

var params struct {
	dnsName      string
	mode         string
	targetDomain string
}

type fakeProducer struct {
	channel      chan *pkg.Endpoint
	mode         string
	dnsName      string
	targetDomain string
}

func init() {
	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").Default(defaultDomain).StringVar(&params.dnsName)
	kingpin.Flag("fake-mode", "The mode to run in.").Default(ipMode).StringVar(&params.mode)
	kingpin.Flag("fake-target-domain", "The target domain for hostname mode.").Default(defaultDomain).StringVar(&params.targetDomain)

	rand.Seed(time.Now().UnixNano())
}

func NewFake() (*fakeProducer, error) {
	return &fakeProducer{
		channel:      make(chan *pkg.Endpoint),
		mode:         params.mode,
		dnsName:      params.dnsName,
		targetDomain: params.targetDomain,
	}, nil
}

func (a *fakeProducer) Endpoints() ([]*pkg.Endpoint, error) {
	endpoints := make([]*pkg.Endpoint, 0)

	for i := 0; i < 10; i++ {
		endpoints = append(endpoints, a.generateEndpoint())
	}

	return endpoints, nil
}

func (a *fakeProducer) StartWatch() error {
	for {
		a.channel <- a.generateEndpoint()
		time.Sleep(5 * time.Second)
	}

	return nil
}

func (a *fakeProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}

func (a *fakeProducer) generateEndpoint() *pkg.Endpoint {
	endpoint := &pkg.Endpoint{
		DNSName: fmt.Sprintf("%s.%s", randomString(2), a.dnsName),
	}

	switch a.mode {
	case ipMode:
		endpoint.IP = net.IPv4(
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
			byte(randomNumber(1, 255)),
		).String()
	case hostnameMode:
		endpoint.Hostname = fmt.Sprintf("%s.%s", randomString(6), a.targetDomain)
	default:
		log.Fatalf("Unknown mode: %s", a.mode)
	}

	return endpoint
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
