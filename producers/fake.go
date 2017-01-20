package producers

import (
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	defaultFakeDomain = "example.org."
	defaultFakeMode   = "ip"
	ipMode            = "ip"
	hostnameMode      = "hostname"
	fixedMode         = "fixed"
)

type fakeProducer struct {
	mode          string
	dnsName       string
	targetDomain  string
	fixedDNSName  string
	fixedIP       string
	fixedHostname string
}

type FakeProducerOptions struct {
	DNSName       string
	Mode          string
	TargetDomain  string
	FixedDNSName  string
	FixedIP       string
	FixedHostname string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewFakeProducer(cfg *FakeProducerOptions) (*fakeProducer, error) {
	if cfg.DNSName == "" {
		cfg.DNSName = defaultFakeDomain
	}
	if cfg.Mode == "" {
		cfg.Mode = defaultFakeMode
	}

	return &fakeProducer{
		mode:          cfg.Mode,
		dnsName:       cfg.DNSName,
		targetDomain:  cfg.TargetDomain,
		fixedDNSName:  cfg.FixedDNSName,
		fixedIP:       cfg.FixedIP,
		fixedHostname: cfg.FixedHostname,
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
	case fixedMode:
		endpoint.DNSName = a.fixedDNSName
		endpoint.IP = a.fixedIP
		endpoint.Hostname = a.fixedHostname
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
