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
	defaultFormat     = "{{.Name}}-{{.Namespace}}"
	annotationKey     = "k8s.zalando.org/dnsname"
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
	allServices, err := a.client.Services(api.NamespaceAll).List(api.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("Unable to retrieve list of services: %v", err)
	}

	log.Debugln("Current services and their endpoints:")
	log.Debugln("=====================================")
	ret := make([]*pkg.Endpoint, 0, len(allServices.Items))
	for _, svc := range allServices.Items {
		if valid, problem := validateService(svc); !valid {
			log.Warnln(problem)
			continue
		}

		ep, err := a.convertToEndpoint(svc)
		if err != nil {
			// TODO: consider allowing the service continue running and just log this error
			return nil, err
		}

		ret = append(ret, ep)
	}

	return ret, nil
}

func (a *kubernetesProducer) StartWatch() error {
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

		if valid, problem := validateService(*svc); !valid {
			log.Println(problem)
			continue
		}

		ep, err := a.convertToEndpoint(*svc)
		if err != nil {
			// TODO: consider letting the service continue running and just log this error
			return err
		}

		a.channel <- ep
	}

	return pkg.ErrEventChannelClosed
}

func (a *kubernetesProducer) ResultChan() (chan *pkg.Endpoint, error) {
	return a.channel, nil
}

func validateService(svc api.Service) (bool, string) {
	switch {
	case len(svc.Status.LoadBalancer.Ingress) == 0:
		return false, fmt.Sprintf(
			"The load balancer of service '%s/%s' does not have any ingress.",
			svc.Namespace, svc.Name,
		)
	case len(svc.Status.LoadBalancer.Ingress) > 1:
		// TODO(linki): in case we have multiple ingress we can just create multiple A or CNAME records
		return false, fmt.Sprintf(
			"Cannot register service '%s/%s'. More than one ingress is not supported",
			svc.Namespace, svc.Name,
		)
	}

	return true, ""
}

func (a *kubernetesProducer) convertToEndpoint(svc api.Service) (*pkg.Endpoint, error) {
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
