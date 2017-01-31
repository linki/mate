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

type kubernetesServiceProducer struct {
	client *k8s.Clientset
	tmpl   *template.Template
}

func NewKubernetesService(cfg *KubernetesOptions) (*kubernetesServiceProducer, error) {
	client, err := kubernetes.NewClient(cfg.APIServer)
	if err != nil {
		return nil, fmt.Errorf("[Service] Unable to setup Kubernetes API client: %v", err)
	}

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(cfg.Format)
	if err != nil {
		return nil, fmt.Errorf("[Service] Error parsing template: %s", err)
	}

	return &kubernetesServiceProducer{
		client: client,
		tmpl:   tmpl,
	}, nil
}

func (a *kubernetesServiceProducer) Endpoints() ([]*pkg.Endpoint, error) {
	allServices, err := a.client.Services(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("[Service] Unable to retrieve list of services: %v", err)
	}

	endpoints := make([]*pkg.Endpoint, 0)

	for _, svc := range allServices.Items {
		if err := validateService(svc); err != nil {
			log.Warnln(err)
			continue
		}

		eps, err := a.convertServiceToEndpoints(svc)
		if err != nil {
			log.Error(err)
			continue
		}

		endpoints = append(endpoints, eps...)
	}

	return endpoints, nil
}

func (a *kubernetesServiceProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

loop:
	for {
		w, err := a.client.Services(api.NamespaceAll).Watch(api.ListOptions{})
		if err != nil {
			errChan <- fmt.Errorf("[Service] Unable to watch list of services: %v", err)

			select {
			case <-done:
				log.Info("[Service] Exited monitoring loop.")
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
					errChan <- fmt.Errorf("[Service] Event listener received an error, terminating: %v", event)
					continue
				}

				if event.Type != watch.Added && event.Type != watch.Modified {
					continue
				}

				svc, ok := event.Object.(*api.Service)
				if !ok {
					// If the object wasn't a Service we can safely ignore it
					log.Printf("[Service] Cannot cast object to service: %v", svc)
					continue
				}

				log.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

				if err := validateService(*svc); err != nil {
					log.Warnln(err)
					continue
				}

				eps, err := a.convertServiceToEndpoints(*svc)
				if err != nil {
					log.Warnln(err)
					continue
				}

				for _, ep := range eps {
					results <- ep
				}
			case <-done:
				log.Info("[Service] Exited monitoring loop.")
				return
			}
		}
	}
}

func validateService(svc api.Service) error {
	if len(svc.Status.LoadBalancer.Ingress) == 0 {
		return fmt.Errorf(
			"[Service] The load balancer of service '%s/%s' does not have any ingress.",
			svc.Namespace, svc.Name,
		)
	}

	return nil
}

func (a *kubernetesServiceProducer) convertServiceToEndpoints(svc api.Service) ([]*pkg.Endpoint, error) {
	dnsName := svc.ObjectMeta.Annotations[annotationKey]

	if dnsName == "" {
		var buf bytes.Buffer
		if err := a.tmpl.Execute(&buf, svc); err != nil {
			return nil, fmt.Errorf("[Service] Error applying template: %s", err)
		}

		dnsName = pkg.SanitizeDNSName(buf.String())
	}

	endpoints := make([]*pkg.Endpoint, 0, len(svc.Status.LoadBalancer.Ingress))

	for _, i := range svc.Status.LoadBalancer.Ingress {
		ep := &pkg.Endpoint{
			DNSName: dnsName,
		}

		ep.IP = i.IP
		ep.Hostname = i.Hostname

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}
