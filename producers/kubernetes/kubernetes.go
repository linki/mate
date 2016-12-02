package kubernetes

import (
	"errors"
	"fmt"
	"log"
	"net/url"

	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	defaultKubeServer = "http://127.0.0.1:8001"
	defaultFormat     = "{{.Name}}-{{.Namespace}}"
	annotationKey     = "zalando.org/dnsname"
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
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").Default(defaultKubeServer).URLVar(&params.kubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries").Default(defaultFormat).StringVar(&params.format)
	kingpin.Flag("kubernetes-domain", "The DNS domain to create DNS entries under.").StringVar(&params.domain)
}

func NewProducer() (*kubernetesProducer, error) {
	if params.domain == "" {
		return nil, errors.New("Please provide --kubernetes-domain")
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
	ret := make([]*pkg.Endpoint, 0)

	ingressEndpoints, err := a.ingress.Endpoints()
	if err != nil {
		log.Fatalf("Error getting endpoints from producer: %v", err)
	}

	serviceEndpoints, err := a.service.Endpoints()
	if err != nil {
		log.Fatalf("Error getting endpoints from producer: %v", err)
	}

	ret = append(ret, ingressEndpoints...)
	ret = append(ret, serviceEndpoints...)

	return ret, nil
}

func (a *kubernetesProducer) StartWatch() error {

	// a.wg.Add(1)

	go func() {
		// defer c.wg.Done()

		for {
			err := a.ingress.StartWatch()
			switch {
			case err == pkg.ErrEventChannelClosed:
				//log.Debugln("Unable to read from channel. Channel was closed. Trying to restart watch...")
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
		// defer c.wg.Done()

		for {
			err := a.service.StartWatch()
			switch {
			case err == pkg.ErrEventChannelClosed:
			//	log.Debugln("Unable to read from channel. Channel was closed. Trying to restart watch...")
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

	return nil
}

func (a *kubernetesProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}
