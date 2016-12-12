package kubernetes

import (
	"errors"
	"fmt"
	"net/url"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	annotationKey = "zalando.org/dnsname"
)

var params struct {
	project string
	zone    string
	domain  string

	kubeServer *url.URL
	format     string
}

type kubernetesProducer struct {
	ingress *kubernetesIngressProducer
	service *kubernetesServiceProducer
	channel chan *pkg.Endpoint
}

func init() {
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&params.kubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&params.format)
}

func NewProducer() (*kubernetesProducer, error) {
	if params.format == "" {
		return nil, errors.New("Please provider --kubernetes-format")
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
		channel: make(chan *pkg.Endpoint),
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

func (a *kubernetesProducer) StartWatch() error {
	go func() {
		for {
			err := a.ingress.StartWatch()
			switch {
			case err == pkg.ErrEventChannelClosed:
				log.Debugln("Unable to read from channel. Channel was closed. Trying to restart watch...")
			case err != nil:
				log.Fatalln(err)
			}
		}
	}()

	ingressChannel, err := a.ingress.ResultChan()
	if err != nil {
		return err
	}

	go func() {
		for {
			err := a.service.StartWatch()
			switch {
			case err == pkg.ErrEventChannelClosed:
				log.Debugln("Unable to read from channel. Channel was closed. Trying to restart watch...")
			case err != nil:
				log.Fatalln(err)
			}
		}
	}()

	serviceChannel, err := a.service.ResultChan()
	if err != nil {
		return err
	}

	for {
		select {
		case event := <-ingressChannel:
			a.channel <- event
		case event := <-serviceChannel:
			a.channel <- event
		}
	}

	return pkg.ErrEventChannelClosed
}

func (a *kubernetesProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}
