package producers

import (
	"errors"
	"fmt"
	"net/url"
	"sync"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/interfaces"
	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/producers/null"
)

const (
	annotationKey = "zalando.org/dnsname"
)

var kubernetesParams struct {
	project string
	zone    string

	kubeServer      *url.URL
	format          string
	enableNodePorts bool
}

type kubernetesProducer struct {
	ingress   interfaces.Producer
	service   interfaces.Producer
	nodePorts interfaces.Producer
}

func init() {
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&kubernetesParams.kubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&kubernetesParams.format)
	kingpin.Flag("enable-node-port-services", "When true, generates DNS entries for type=NodePort services").BoolVar(&kubernetesParams.enableNodePorts)
}

func NewKubernetes() (*kubernetesProducer, error) {
	if kubernetesParams.format == "" {
		return nil, errors.New("Please provide --kubernetes-format")
	}

	var err error

	producer := &kubernetesProducer{}

	producer.ingress, err = NewKubernetesIngress()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	producer.service, err = NewKubernetesService()
	if err != nil {
		return nil, fmt.Errorf("[Kubernetes] Error creating producer: %v", err)
	}

	if kubernetesParams.enableNodePorts {
		producer.nodePorts, err = NewKubernetesNodePorts()
	} else {
		producer.nodePorts, err = null.NewNull()
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
