package producers

import (
	"fmt"
	"math/rand"
	"net"
	"time"

	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/pkg"
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
		endpoint := &pkg.Endpoint{
			DNSName: fmt.Sprintf("%s.%s", randomString(2), params.dnsName),
			IP: net.IPv4(
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
			).String(),
		}
		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func (a *fakeProducer) StartWatch() error {
	for {
		a.channel <- &pkg.Endpoint{
			DNSName: fmt.Sprintf("%s.%s", randomString(2), params.dnsName),
			IP: net.IPv4(
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
				byte(randomNumber(1, 255)),
			).String(),
		}

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
