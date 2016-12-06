package kubernetes

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"

	log "github.com/Sirupsen/logrus"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"

	"github.com/zalando-incubator/mate/pkg"
)

type kubernetesServiceProducer struct {
	client  *unversioned.Client
	tmpl    *template.Template
	channel chan *pkg.Endpoint
}

func NewKubernetesService() (*kubernetesServiceProducer, error) {
	if params.domain == "" {
		return nil, errors.New("Please provide --kubernetes-domain")
	}

	client, err := unversioned.New(&restclient.Config{
		Host: params.kubeServer.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to create Kubernetes API client: %v", err)
	}

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(params.format)
	if err != nil {
		return nil, fmt.Errorf("Error parsing template: %s", err)
	}

	return &kubernetesServiceProducer{
		client:  client,
		tmpl:    tmpl,
		channel: make(chan *pkg.Endpoint),
	}, nil
}

func (a *kubernetesServiceProducer) Endpoints() ([]*pkg.Endpoint, error) {
	allServices, err := a.client.Services(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve list of services: %v", err)
	}

	endpoints := make([]*pkg.Endpoint, 0)

	for _, svc := range allServices.Items {
		if err := validateService(svc); err != nil {
			log.Warnln(err)
			continue
		}

		ep, err := a.convertServiceToEndpoint(svc)
		if err != nil {
			// TODO: consider allowing the service continue running and just log this error
			return nil, err
		}

		endpoints = append(endpoints, ep)
	}

	return endpoints, nil
}

func (a *kubernetesServiceProducer) StartWatch() error {
	w, err := a.client.Services(api.NamespaceAll).Watch(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Unable to watch list of services: %v", err)
	}

	for event := range w.ResultChan() {
		if event.Type == watch.Error {
			// TODO: consider allowing the service continue running and just log this error
			return fmt.Errorf("Event listener received an error, terminating: %v", event)
		}

		if event.Type != watch.Added && event.Type != watch.Modified {
			continue
		}

		svc, ok := event.Object.(*api.Service)
		if !ok {
			// If the object wasn't a Service we can safely ignore it
			log.Printf("Cannot cast object to service: %v", svc)
			continue
		}

		log.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

		if err := validateService(*svc); err != nil {
			log.Warnln(err)
			continue
		}

		ep, err := a.convertServiceToEndpoint(*svc)
		if err != nil {
			// TODO: consider letting the service continue running and just log this error
			return err
		}

		a.channel <- ep
	}

	return pkg.ErrEventChannelClosed
}

func (a *kubernetesServiceProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}

func validateService(svc api.Service) error {
	switch {
	case len(svc.Status.LoadBalancer.Ingress) == 0:
		return fmt.Errorf(
			"The load balancer of service '%s/%s' does not have any ingress.",
			svc.Namespace, svc.Name,
		)
	case len(svc.Status.LoadBalancer.Ingress) > 1:
		// TODO(linki): in case we have multiple ingress we can just create multiple A or CNAME records
		return fmt.Errorf(
			"Cannot register service '%s/%s'. More than one ingress is not supported",
			svc.Namespace, svc.Name,
		)
	}

	return nil
}

func (a *kubernetesServiceProducer) convertServiceToEndpoint(svc api.Service) (*pkg.Endpoint, error) {
	ep := &pkg.Endpoint{
		DNSName: svc.ObjectMeta.Annotations[annotationKey],
	}

	if ep.DNSName == "" {
		var buf bytes.Buffer
		if err := a.tmpl.Execute(&buf, svc); err != nil {
			return nil, fmt.Errorf("Error applying template: %s", err)
		}

		ep.DNSName = fmt.Sprintf("%s.%s", buf.String(), params.domain)
	}

	for _, i := range svc.Status.LoadBalancer.Ingress {
		ep.IP = i.IP
		ep.Hostname = i.Hostname

		// take the first entry and exit
		// TODO(linki): we could easily return a list of endpoints
		break
	}

	return ep, nil
}
