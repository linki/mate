package producers

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	defaultDomain = "example.org."
	ipMode        = "ip"
	hostnameMode  = "hostname"
)

var fakeParams struct {
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
	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").Default(defaultDomain).StringVar(&fakeParams.dnsName)
	kingpin.Flag("fake-mode", "The mode to run in.").Default(ipMode).StringVar(&fakeParams.mode)
	kingpin.Flag("fake-target-domain", "The target domain for hostname mode.").Default(defaultDomain).StringVar(&fakeParams.targetDomain)

	rand.Seed(time.Now().UnixNano())
}

func NewFake() (*fakeProducer, error) {
	return &fakeProducer{
		channel:      make(chan *pkg.Endpoint),
		mode:         fakeParams.mode,
		dnsName:      fakeParams.dnsName,
		targetDomain: fakeParams.targetDomain,
	}, nil
}

func (a *fakeProducer) Endpoints() ([]*pkg.Endpoint, error) {
	endpoints := make([]*pkg.Endpoint, 0)

	for i := 0; i < 10; i++ {
		endpoint, err := a.generateEndpoint()
		if err != nil {
			log.Warn("[Fake] Error generating fake endpoint: %v", err)
			continue
		}

		endpoints = append(endpoints, endpoint)
	}

	return endpoints, nil
}

func (a *fakeProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	for {
		select {
		case <-time.After(5 * time.Second):
		case <-done:
			log.Info("[Fake] Exited monitoring loop.")
			return
		}

		endpoint, err := a.generateEndpoint()
		if err != nil {
			errChan <- err
			continue
		}

		results <- endpoint
	}
}

func (a *fakeProducer) generateEndpoint() (*pkg.Endpoint, error) {
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
		return nil, fmt.Errorf("Unknown mode: %s", a.mode)
	}

	return endpoint, nil
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
