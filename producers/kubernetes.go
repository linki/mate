package producers

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"

	"github.bus.zalan.do/teapot/mate/pkg"
	"github.bus.zalan.do/teapot/pkg/kubernetes"
)

const (
	defaultKubeServer = "http://127.0.0.1:8001"
	defaultFormat     = "{{.Namespace}}-{{.Name}}"
)

type kubernetesProducer struct {
	client  *unversioned.Client
	tmpl    *template.Template
	channel chan *pkg.Endpoint
}

func init() {
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").Default(defaultKubeServer).URLVar(&params.kubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries").Default(defaultFormat).StringVar(&params.format)
	kingpin.Flag("kubernetes-domain", "The DNS domain to create DNS entries under.").StringVar(&params.domain)
}

func NewKubernetes() (*kubernetesProducer, error) {
	if params.domain == "" {
		return nil, errors.New("Please provide --kubernetes-domain")
	}

	client := kubernetes.NewHealthyClient(params.kubeServer)

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(params.format)
	if err != nil {
		return nil, fmt.Errorf("Error parsing template: %s", err)
	}

	return &kubernetesProducer{
		client:  client,
		tmpl:    tmpl,
		channel: make(chan *pkg.Endpoint),
	}, nil
}

func (a *kubernetesProducer) Endpoints() ([]*pkg.Endpoint, error) {
	allServices, err := a.client.Services(api.NamespaceAll).List(api.ListOptions{
	// better to filter by services that have external IPs here
	// FieldSelector: fields.OneTermEqualSelector("external ip", "exists"),
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve list of services: %v", err)
	}

	services := make([]api.Service, 0, len(allServices.Items))

	for _, svc := range allServices.Items {
		if len(svc.Status.LoadBalancer.Ingress) == 1 {
			services = append(services, svc)
		}

		// TODO: support multiple IPs
		if len(svc.Status.LoadBalancer.Ingress) > 1 {
			log.Warnf("Cannot register service '%s/%s'. More than one ingress is not supported",
				svc.Namespace, svc.Name)
		}
	}

	log.Debugln("Current services and their endpoints:")
	log.Debugln("=====================================")
	for _, svc := range services {
		log.Debugln(svc.Name, svc.Namespace, svc.Status.LoadBalancer.Ingress[0])
	}

	ret := make([]*pkg.Endpoint, 0, len(services))

	for _, svc := range services {
		var buf bytes.Buffer

		err = a.tmpl.Execute(&buf, svc)
		if err != nil {
			return nil, fmt.Errorf("Error applying template: %s", err)
		}

		endpoint := &pkg.Endpoint{
			DNSName: fmt.Sprintf("%s.%s", buf.String(), params.domain),
			// TODO: allow multiple IPs
			IP: svc.Status.LoadBalancer.Ingress[0].IP,
		}

		ret = append(ret, endpoint)
	}

	return ret, nil
}

func (a *kubernetesProducer) StartWatch() error {
	w, err := a.client.Services(api.NamespaceAll).Watch(api.ListOptions{})
	if err != nil {
		return fmt.Errorf("Unable to watch list of services: %v", err)
	}

	events := w.ResultChan()

	for {
		event, ok := <-events
		if !ok {
			// The channel closes regularly in which case we restart the watch
			return fmt.Errorf("Unable to read from channel. Channel was closed. Trying to restart watch...")

		}
		if event.Type == watch.Added || event.Type == watch.Modified {
			svc, ok := event.Object.(*api.Service)
			if !ok {
				// If the object wasn't a Service we can safely ignore it
				log.Printf("Cannot cast object to service: %v", svc)
				continue
			}

			log.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

			if len(svc.Status.LoadBalancer.Ingress) > 1 {
				// Print a warning about the ignored service, no need to crash here
				log.Printf("Cannot register service '%s/%s'. More than one ingress is not supported",
					svc.Namespace, svc.Name)
				continue
			}

			if len(svc.Status.LoadBalancer.Ingress) == 1 {
				var buf bytes.Buffer

				err = a.tmpl.Execute(&buf, svc)
				if err != nil {
					return fmt.Errorf("Error applying template: %s", err)
				}

				a.channel <- &pkg.Endpoint{
					DNSName: fmt.Sprintf("%s.%s", buf.String(), params.domain),
					// allow multiple IPs
					IP: svc.Status.LoadBalancer.Ingress[0].IP,
				}
			}
		}

		if event.Type == watch.Deleted {
			// can be ignored due to sync
		}

		if event.Type == watch.Error {
			return fmt.Errorf("Event listener received an error, terminating: %v", event)
		}
	}

	return nil
}

func (a *kubernetesProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}
