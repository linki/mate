package producers

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/pkg"
)

const (
	annotationKey = "zalando.org/dnsname"
)

type kubernetesProducer struct {
	ingress   Producer
	service   Producer
	nodePorts Producer
}

type KubernetesOptions struct {
	APIServer      *url.URL
	Format         string
	TrackNodePorts bool
	Filter         map[string]string
}

func NewKubernetesProducer(cfg *KubernetesOptions) (*kubernetesProducer, error) {
	if cfg.Format == "" {
		return nil, errors.New("Please provide --kubernetes-format")
	}

	if cfg.TrackNodePorts {
		log.Infof("Please note, creating DNS entries for NodePort services doesn't currently work in combination with the AWS consumer.")
	}

	var err error

	producer := &kubernetesProducer{}

	producer.ingress, err = NewKubernetesIngress(cfg)
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	producer.service, err = NewKubernetesService(cfg)
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	if cfg.TrackNodePorts {
		producer.nodePorts, err = NewKubernetesNodePorts(cfg)
	} else {
		producer.nodePorts, err = NewNullProducer()
	}
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	return producer, nil
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

	nodePortsEndpoints, err := a.nodePorts.Endpoints()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error getting endpoints from producer: %v", err)
	}

	ingressEndpoints = append(ingressEndpoints, serviceEndpoints...)
	return append(ingressEndpoints, nodePortsEndpoints...), nil
}

func (a *kubernetesProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	go a.ingress.Monitor(results, errChan, done, wg)
	go a.service.Monitor(results, errChan, done, wg)
	go a.nodePorts.Monitor(results, errChan, done, wg)

	<-done
	log.Info("[Kubernetes] Exited monitoring loop.")
}
