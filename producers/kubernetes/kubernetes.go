package kubernetes

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

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

	kubeServer *url.URL
	format     string
}

type kubernetesProducer struct {
	ingress *kubernetesIngressProducer
	service *kubernetesServiceProducer
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
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	service, err := NewKubernetesService()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	return &kubernetesProducer{
		ingress: ingress,
		service: service,
	}, nil
}

func (a *kubernetesProducer) Endpoints() ([]*pkg.Endpoint, error) {
	ingressEndpoints, err := a.ingress.Endpoints()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error getting endpoints from producer: %v", err)
	}

	serviceEndpoints, err := a.service.Endpoints()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error getting endpoints from producer: %v", err)
	}

	return append(ingressEndpoints, serviceEndpoints...), nil
}

func (a *kubernetesProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	go a.ingress.Monitor(results, errChan, done, wg)
	go a.service.Monitor(results, errChan, done, wg)

	<-done
	log.Info("[Kubernetes] Exited monitoring loop.")
}
