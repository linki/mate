package kubernetes

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	annotationKey = "zalando.org/dnsname"
)

var params struct {
	project string
	zone    string

	kubeServer *url.URL
	format     string
}

type kubernetesProducer struct {
	ingress *kubernetesIngressProducer
	service *kubernetesServiceProducer

	wg sync.WaitGroup
}

func init() {
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&params.kubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&params.format)
}

func NewProducer() (*kubernetesProducer, error) {
	if params.format == "" {
		return nil, errors.New("Please provide --kubernetes-format")
	}

	ingress, err := NewKubernetesIngress()
	if err != nil {
		return nil, fmt.Errorf("Error creating producer: %v", err)
	}

	service, err := NewKubernetesService()
	if err != nil {
		return nil, fmt.Errorf("Error creating producer: %v", err)
	}

	return &kubernetesProducer{
		ingress: ingress,
		service: service,
	}, nil
}

func (a *kubernetesProducer) Endpoints() ([]*pkg.Endpoint, error) {
	ingressEndpoints, err := a.ingress.Endpoints()
	if err != nil {
		return nil, fmt.Errorf("Error getting endpoints from producer: %v", err)
	}

	serviceEndpoints, err := a.service.Endpoints()
	if err != nil {
		return nil, fmt.Errorf("Error getting endpoints from producer: %v", err)
	}

	return append(ingressEndpoints, serviceEndpoints...), nil
}

func (a *kubernetesProducer) Monitor() (chan *pkg.Endpoint, chan error) {
	channel1, errors1 := a.ingress.Monitor()
	channel2, errors2 := a.service.Monitor()

	channel := make(chan *pkg.Endpoint)
	errors := make(chan error)

	a.wg.Add(1)

	go func() {
		defer a.wg.Done()

		for {
			select {
			case event := <-channel1:
				channel <- event
			case event := <-channel2:
				channel <- event
			case event := <-errors1:
				errors <- event
			case event := <-errors2:
				errors <- event
			}
		}
	}()

	return channel, errors
}
