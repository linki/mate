package providers

import (
	"bytes"
	"fmt"
	"html/template"
	"strings"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/unversioned"

	"github.bus.zalan.do/teapot/mate/pkg/kubernetes"
)

const (
	defaultKubeServer = "http://127.0.0.1:8080"
	defaultFormat     = "{{.Namespace}}-{{.Name}}"
)

type kubernetesProvider struct {
	client *unversioned.Client
	tmpl   *template.Template
}

// var params struct {
// 	kubeServer string
// 	format     string
// }

func init() {
	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").Default(defaultKubeServer).StringVar(&params.kubeServer)
	kingpin.Flag("format", "Format of DNS entries").Default(defaultFormat).StringVar(&params.format)
}

func (a *kubernetesProvider) Endpoints() ([]*Endpoint, error) {
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

		// TODO: why not?
		if len(svc.Status.LoadBalancer.Ingress) > 1 {
			// Print a warning about the ignored service, no need to crash here
			log.Warnf("Cannot register service '%s/%s'. More than one ingress is not supported",
				svc.Namespace, svc.Name)
		}
	}

	log.Debugln("Current services and their endpoints:")
	log.Debugln("=====================================")
	for _, svc := range services {
		log.Debugln(svc.Name, svc.Namespace, svc.Status.LoadBalancer.Ingress[0])
	}

	ret := make([]*Endpoint, 0, len(services))

	for _, svc := range services {
		var buf bytes.Buffer

		err = a.tmpl.Execute(&buf, svc)
		if err != nil {
			return nil, fmt.Errorf("Error applying template: %s", err)
		}

		ret = append(ret, &Endpoint{
			DNSName: buf.String(),
			IP:      svc.Status.LoadBalancer.Ingress[0].IP,
		})
	}

	return ret, nil
}

func NewKubernetesProvider() (*kubernetesProvider, error) {
	client := kubernetes.NewHealthyClient(params.kubeServer)

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(params.format)
	if err != nil {
		return nil, fmt.Errorf("Error parsing template: %s", err)
	}

	return &kubernetesProvider{
		client: client,
		tmpl:   tmpl,
	}, nil
}
