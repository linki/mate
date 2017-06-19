package producers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	api "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/watch"

	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/pkg/kubernetes"
	k8s "k8s.io/client-go/kubernetes"
)

type kubernetesNodePortsProducer struct {
	client *k8s.Clientset
	tmpl   *template.Template
}

func NewKubernetesNodePorts(cfg *KubernetesOptions) (*kubernetesNodePortsProducer, error) {
	client, err := kubernetes.NewClient(cfg.APIServer)
	if err != nil {
		return nil, fmt.Errorf("Unable to setup Kubernetes API client: %v", err)
	}

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(cfg.Format)
	if err != nil {
		return nil, fmt.Errorf("Error parsing template: %s", err)
	}

	return &kubernetesNodePortsProducer{
		client: client,
		tmpl:   tmpl,
	}, nil
}

func (a *kubernetesNodePortsProducer) Endpoints() ([]*pkg.Endpoint, error) {
	allServices, err := a.client.Services(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("[NodePort] Unable to retrieve list of services: %v", err)
	}

	endpoints := make([]*pkg.Endpoint, 0)

	for _, svc := range allServices.Items {
		if err := validateNodePortService(svc); err != nil {
			log.Warnln(err)
			continue
		}

		eps, err := a.convertNodePortServiceToEndpoint(svc)
		if err != nil {
			log.Error(err)
			continue
		}

		endpoints = append(endpoints, eps...)
	}

	return endpoints, nil
}

func (a *kubernetesNodePortsProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

loop:
	for {
		w, err := a.client.Services(api.NamespaceAll).Watch(api.ListOptions{})
		if err != nil {
			errChan <- fmt.Errorf("[NodePort] Unable to watch list of services: %v", err)

			select {
			case <-done:
				log.Info("[NodePort] Exited monitoring loop.")
				return
			case <-time.After(5 * time.Second):
				goto loop
			}
		}

		for {
			select {
			case event, ok := <-w.ResultChan():
				if !ok {
					goto loop
				}

				if event.Type == watch.Error {
					// TODO: consider allowing the service continue running and just log this error
					errChan <- fmt.Errorf("[NodePort] Event listener received an error, terminating: %v", event)
					continue
				}

				if event.Type != watch.Added && event.Type != watch.Modified {
					continue
				}

				svc, ok := event.Object.(*api.Service)
				if !ok {
					// If the object wasn't a Service we can safely ignore it
					log.Printf("[NodePort] Cannot cast object to service: %v", svc)
					continue
				}

				log.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

				if err := validateNodePortService(*svc); err != nil {
					log.Warnln(err)
					continue
				}

				eps, err := a.convertNodePortServiceToEndpoint(*svc)
				if err != nil {
					log.Warnln(err)
					continue
				}

				for _, ep := range eps {
					results <- ep
				}
			case <-done:
				log.Info("[NodePort] Exited monitoring loop.")
				return
			}
		}
	}
}

func validateNodePortService(svc api.Service) error {
	if svc.Spec.Type != api.ServiceTypeNodePort {
		return fmt.Errorf("Not a node port service: %s (%s)", svc.Name, svc.Spec.Type)
	}

	return nil
}

func (a *kubernetesNodePortsProducer) convertNodePortServiceToEndpoint(svc api.Service) ([]*pkg.Endpoint, error) {
	nodes := a.getNodes()

	endpoints := make([]*pkg.Endpoint, 0, len(nodes))

	for _, node := range nodes {
		for _, address := range node.Status.Addresses {
			if address.Type != api.NodeExternalIP {
				log.Debugf("%s address: %s (%s) is not an external IP", node.Name, address.Address, address.Type)
				continue
			}

			for _, dnsname := range strings.Split(svc.ObjectMeta.Annotations[annotationKey], ",") {
				ep := &pkg.Endpoint{
					DNSName: dnsname,
				}

				if ep.DNSName == "" {
					var buf bytes.Buffer
					if err := a.tmpl.Execute(&buf, svc); err != nil {
						return nil, fmt.Errorf("Error applying template: %s", err)
					}

					ep.DNSName = pkg.SanitizeDNSName(buf.String())
				}

				log.Debugf("%s address: %s (%s)", node.Name, address.Address, address.Type)

				ep.IP = address.Address

				endpoints = append(endpoints, ep)
			}
		}
	}

	return endpoints, nil
}

func (a *kubernetesNodePortsProducer) getNodes() []api.Node {
	nodes, err := a.client.Nodes().List(api.ListOptions{})
	if err != nil {
		log.Fatalf("Unable to retrieve list of nodes: %v", err)
	}

	return nodes.Items
}
