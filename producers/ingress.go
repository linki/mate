package producers

import (
	"fmt"
	"html/template"
	"strings"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/pkg/kubernetes"
	k8s "k8s.io/client-go/kubernetes"
	api "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
	"k8s.io/client-go/pkg/watch"
)

type kubernetesIngressProducer struct {
	client *k8s.Clientset
	tmpl   *template.Template
	filter map[string]string
}

func NewKubernetesIngress(cfg *KubernetesOptions) (*kubernetesIngressProducer, error) {
	client, err := kubernetes.NewClient(cfg.APIServer)
	if err != nil {
		return nil, fmt.Errorf("[Ingress] Unable to setup Kubernetes API client: %v", err)
	}

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(cfg.Format)
	if err != nil {
		return nil, fmt.Errorf("[Ingress] Error parsing template: %s", err)
	}

	return &kubernetesIngressProducer{
		client: client,
		tmpl:   tmpl,
		filter: cfg.Filter,
	}, nil
}

func (a *kubernetesIngressProducer) Endpoints() ([]*pkg.Endpoint, error) {
	allIngress, err := a.client.Ingresses(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("[Ingress] Unable to retrieve list of ingress: %v", err)
	}

	endpoints := make([]*pkg.Endpoint, 0)

	for _, ing := range allIngress.Items {
		if err := validateIngress(ing, a.filter); err != nil {
			log.Warnln(err)
			continue
		}

		eps := a.convertIngressToEndpoint(ing)

		endpoints = append(endpoints, eps...)
	}

	return endpoints, nil
}

func (a *kubernetesIngressProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

loop:
	for {
		w, err := a.client.Ingresses(api.NamespaceAll).Watch(api.ListOptions{})
		if err != nil {
			errChan <- fmt.Errorf("[Ingress] Unable to watch list of ingress: %v", err)
			select {
			case <-done:
				log.Info("[Ingress] Exited monitoring loop.")
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
					errChan <- fmt.Errorf("[Ingress] Event listener received an error, terminating: %v", event)
					continue
				}

				if event.Type != watch.Added && event.Type != watch.Modified {
					continue
				}

				ing, ok := event.Object.(*extensions.Ingress)
				if !ok {
					// If the object wasn't a Service we can safely ignore it
					log.Printf("[Ingress] Cannot cast object to ingress: %v", ing)
					continue
				}

				log.Printf("%s: %s/%s", event.Type, ing.Namespace, ing.Name)

				if err := validateIngress(*ing, a.filter); err != nil {
					log.Warnln(err)
					continue
				}

				eps := a.convertIngressToEndpoint(*ing)

				for _, ep := range eps {
					results <- ep
				}
			case <-done:
				log.Info("[Ingress] Exited monitoring loop.")
				return
			}
		}
	}
}

func validateIngress(ing extensions.Ingress, filter map[string]string) error {
	for key := range filter {
		if ing.Annotations[key] != filter[key] {
			return fmt.Errorf(
				"[Ingress] Ingress '%s/%s' doesn't match filter for annotation %s: %s != %s",
				ing.Namespace, ing.Name, key, filter[key], ing.Annotations[key],
			)
		}
	}

	switch {
	case len(ing.Status.LoadBalancer.Ingress) == 0:
		return fmt.Errorf(
			"[Ingress] The load balancer field of ingress '%s/%s' is empty",
			ing.Namespace, ing.Name,
		)
	case len(ing.Status.LoadBalancer.Ingress) > 1:
		// TODO(linki): in case we have multiple ingress we can just create multiple A or CNAME records
		log.Warnf("[Ingress] Ingress '%s/%s' has more than one ingress (%d). Only using the first one.",
			ing.Namespace, ing.Name, len(ing.Status.LoadBalancer.Ingress))
		return nil
	}

	return nil
}

func (a *kubernetesIngressProducer) convertIngressToEndpoint(ing extensions.Ingress) []*pkg.Endpoint {
	endpoints := make([]*pkg.Endpoint, 0, len(ing.Spec.Rules))

	for _, rule := range ing.Spec.Rules {
		ep := &pkg.Endpoint{}

		for _, i := range ing.Status.LoadBalancer.Ingress {
			ep.IP = i.IP
			ep.Hostname = i.Hostname

			// take the first entry and exit
			// TODO(linki): we could easily return a list of endpoints
			break
		}

		ep.DNSName = pkg.SanitizeDNSName(rule.Host)

		endpoints = append(endpoints, ep)
	}

	return endpoints
}
