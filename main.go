package main

import (
	"bytes"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	dns "google.golang.org/api/dns/v1"

	"gopkg.in/alecthomas/kingpin.v2"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

type Endpoint struct {
	Domain    string
	Namespace string
	Name      string
}

const (
	defaultDomain = "oolong.gcp.zalan.do"
	defaultMaster = "http://127.0.0.1:8080"
	defaultFormat = "{{.Namespace}}-{{.Name}}.{{.Domain}}."
)

var (
	master  = kingpin.Flag("master", "The address of the Kubernetes API server.").Default(defaultMaster).String()
	domain  = kingpin.Flag("domain", "The DNS domain by which cluster endpoints are reachable.").Default(defaultDomain).String()
	project = kingpin.Flag("project", "Project ID that manages the zone").Required().String()
	zone    = kingpin.Flag("zone", "Name of the zone to manage.").String()
	format  = kingpin.Flag("format", "Format of DNS entries").Default(defaultFormat).String()

	// version is injected at build-time
	version = "Unknown"
)

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	logger := log.New(os.Stdout, "", log.LstdFlags)

	if *zone == "" {
		*zone = strings.Replace(*domain, ".", "-", -1)
		logger.Printf("No --zone provided, inferring from domain: '%s'.", *zone)
	}

	tmpl, err := template.New("endpoint").Funcs(template.FuncMap{
		"trimPrefix": strings.TrimPrefix,
	}).Parse(*format)
	if err != nil {
		logger.Fatalf("Error parsing template: %s", err)
	}

	config := &restclient.Config{
		Host: *master,
	}
	client, err := client.New(config)
	if err != nil {
		logger.Fatalf("Unable to connect to API server: %v", err)
	}

	allServices, err := client.Services(api.NamespaceAll).List(api.ListOptions{
	// better to filter by services that have external IPs here
	// FieldSelector: fields.OneTermEqualSelector("external ip", "exists"),
	})
	if err != nil {
		logger.Fatalf("Unable to retrieve list of services: %v", err)
	}

	services := make([]api.Service, 0, len(allServices.Items))

	for _, svc := range allServices.Items {
		if len(svc.Status.LoadBalancer.Ingress) == 1 {
			services = append(services, svc)
		}

		if len(svc.Status.LoadBalancer.Ingress) > 1 {
			// Print a warning about the ignored service, no need to crash here
			logger.Printf("Cannot register service '%s/%s'. More than one ingress is not supported",
				svc.Namespace, svc.Name)
		}
	}

	logger.Println("Current services and their endpoints:")
	logger.Println("=====================================")
	for _, svc := range services {
		logger.Println(svc.Name, svc.Namespace, svc.Status.LoadBalancer.Ingress[0])
	}

	dnsClient, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		logger.Fatalf("Unable to get DNS client: %v", err)
	}

	dnsService, err := dns.New(dnsClient)
	if err != nil {
		logger.Fatalf("Unable to create DNS client: %v", err)
	}

	resp, err := dnsService.ResourceRecordSets.List(*project, *zone).Do()
	if err != nil {
		logger.Fatalf("Unable to retrieve resource record sets of %s/%s: %v", *project, *zone, err)
	}

	records := make([]*dns.ResourceRecordSet, 0, len(resp.Rrsets))

	for _, r := range resp.Rrsets {
		if r.Type == "A" && strings.Contains(r.Name, *domain) {
			records = append(records, r)
		}
	}

	logger.Println("Current A records and where they point to:")
	logger.Println("==========================================")
	for _, r := range records {
		logger.Println(r.Name, r.Type, r.Rrdatas)
	}

	change := new(dns.Change)

	for _, svc := range services {
		var buf bytes.Buffer

		endpoint := Endpoint{
			Domain:    *domain,
			Namespace: svc.Namespace,
			Name:      svc.Name,
		}

		err = tmpl.Execute(&buf, endpoint)
		if err != nil {
			logger.Fatalf("Error applying template: %s", err)
		}

		change.Additions = append(change.Additions, &dns.ResourceRecordSet{
			Name:    buf.String(),
			Rrdatas: []string{svc.Status.LoadBalancer.Ingress[0].IP},
			Ttl:     300,
			Type:    "A",
		})
	}

	for _, r := range records {
		change.Deletions = append(change.Deletions, &dns.ResourceRecordSet{
			Name:    r.Name,
			Rrdatas: r.Rrdatas,
			Ttl:     r.Ttl,
			Type:    r.Type,
		})
	}

	_, err = dnsService.Changes.Create(*project, *zone, change).Do()
	if err != nil {
		logger.Fatalf("Unable to create change for %s/%s: %v", *project, *zone, err)
	}

	servicesWatch, err := client.Services(api.NamespaceAll).Watch(api.ListOptions{
	// better to filter by services that have external IPs here
	// FieldSelector: fields.OneTermEqualSelector("external ip", "exists"),
	})
	if err != nil {
		logger.Fatalf("Unable to watch list of services: %v", err)
	}

	events := servicesWatch.ResultChan()

	for {
		event, ok := <-events
		if !ok {
			// If the channel was closed something unexpected happened, let's fail.
			logger.Fatalf("Unable to read from channel. Channel was closed.")
		}

		if event.Type == watch.Added || event.Type == watch.Modified {
			svc, ok := event.Object.(*api.Service)
			if !ok {
				// If the object wasn't a Service we can safely ignore it
				logger.Printf("Cannot cast object to service: %v", svc)
				continue
			}

			logger.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

			if len(svc.Status.LoadBalancer.Ingress) > 1 {
				// Print a warning about the ignored service, no need to crash here
				logger.Printf("Cannot register service '%s/%s'. More than one ingress is not supported",
					svc.Namespace, svc.Name)
				continue
			}

			if len(svc.Status.LoadBalancer.Ingress) == 1 {
				var buf bytes.Buffer

				endpoint := Endpoint{
					Domain:    *domain,
					Namespace: svc.Namespace,
					Name:      svc.Name,
				}

				err = tmpl.Execute(&buf, endpoint)
				if err != nil {
					logger.Fatalf("Error applying template: %s", err)
				}

				change := &dns.Change{
					Additions: []*dns.ResourceRecordSet{
						&dns.ResourceRecordSet{
							Name:    buf.String(),
							Rrdatas: []string{svc.Status.LoadBalancer.Ingress[0].IP},
							Ttl:     300,
							Type:    "A",
						},
					},
				}

				time.Sleep(100 * time.Millisecond)

				_, err := dnsService.Changes.Create(*project, *zone, change).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "alreadyExists") {
						logger.Fatalf("Unable to create change for %s/%s: %v", *project, *zone, err)
					}
				}
			}
		}

		if event.Type == watch.Deleted {
			svc := event.Object.(*api.Service)
			if !ok {
				// If the object wasn't a Service we can safely ignore it
				logger.Printf("Cannot cast object to service: %v", svc)
				continue
			}

			logger.Printf("%s: %s/%s", event.Type, svc.Namespace, svc.Name)

			if len(svc.Status.LoadBalancer.Ingress) > 1 {
				// Print a warning about the ignored service, no need to crash here
				logger.Printf("Cannot deregister service '%s/%s'. More than one ingress is not supported",
					svc.Namespace, svc.Name)
				continue
			}

			if len(svc.Status.LoadBalancer.Ingress) == 1 {
				var buf bytes.Buffer

				endpoint := Endpoint{
					Domain:    *domain,
					Namespace: svc.Namespace,
					Name:      svc.Name,
				}

				err = tmpl.Execute(&buf, endpoint)
				if err != nil {
					logger.Fatalf("Error applying template: %s", err)
				}

				change := &dns.Change{
					Deletions: []*dns.ResourceRecordSet{
						&dns.ResourceRecordSet{
							Name:    buf.String(),
							Rrdatas: []string{svc.Status.LoadBalancer.Ingress[0].IP},
							Ttl:     300,
							Type:    "A",
						},
					},
				}

				time.Sleep(100 * time.Millisecond)

				_, err := dnsService.Changes.Create(*project, *zone, change).Do()
				if err != nil {
					if !strings.Contains(err.Error(), "notFound") {
						logger.Fatalf("Unable to create change for %s/%s: %v", *project, *zone, err)
					}
				}
			}
		}

		if event.Type == watch.Error {
			logger.Fatalf("Event listener received an error, terminating: %v", event)
		}
	}
}
