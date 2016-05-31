package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	dns "google.golang.org/api/dns/v1"

	"k8s.io/kubernetes/pkg/api"
	client "k8s.io/kubernetes/pkg/client/unversioned"
	"k8s.io/kubernetes/pkg/watch"
)

func main() {
	fmt.Println("Starting...")

	client, err := client.NewInCluster()
	// or
	// config := &restclient.Config{
	// 	 Host: "http://127.0.0.1:8080",
	// }
	// client, err := client.New(config)
	if err != nil {
		fmt.Printf("Unable to connect to API server: %v", err)
	}
	services, err := client.Services(api.NamespaceAll).List(api.ListOptions{
	// better to filter by services that have external IPs here
	// FieldSelector: fields.OneTermEqualSelector("external ip", "exists"),
	})
	if err != nil {
		fmt.Printf("Unable to retrieve list of services: %v", err)
	}

	extServices := make([]api.Service, 0, len(services.Items))

	for _, svc := range services.Items {
		if len(svc.Status.LoadBalancer.Ingress) == 1 {
			extServices = append(extServices, svc)
		}

		if len(svc.Status.LoadBalancer.Ingress) > 1 {
			fmt.Print("more than one ingress not supported")
			os.Exit(1)
		}
	}

	fmt.Println("Current services and their endpoints:")
	for _, svc := range extServices {
		fmt.Println(svc.Name, svc.Namespace, svc.Status.LoadBalancer.Ingress[0])
	}

	dnsClient, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		fmt.Printf("Unable to get default client: %v", err)
		os.Exit(1)
	}

	dnsService, err := dns.New(dnsClient)

	resp2, err := dnsService.ResourceRecordSets.List("zalando-teapot", "gcp-zalan-do").Do()
	if err != nil {
		fmt.Printf("Unable to retrieve resource record sets of zalando-teapot/gcp-zalan-do: %v", err)
		os.Exit(1)
	}

	records := make([]*dns.ResourceRecordSet, 0, len(resp2.Rrsets))

	for _, r := range resp2.Rrsets {
		if r.Type == "A" && strings.Contains(r.Name, "oolong") {
			records = append(records, r)
		}
	}

	fmt.Println("Current A records and where they point to:")
	for _, r := range records {
		fmt.Println(r.Name, r.Type, r.Rrdatas)
	}

	change := &dns.Change{}

	for _, svc := range extServices {
		change.Additions = append(change.Additions, &dns.ResourceRecordSet{
			Name:    fmt.Sprintf("%s-%s.oolong.gcp.zalan.do.", svc.Namespace, svc.Name),
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

	// spew.Dump(change)

	_, err = dnsService.Changes.Create("zalando-teapot", "gcp-zalan-do", change).Do()
	if err != nil {
		fmt.Printf("Unable to create change for zalando-teapot/gcp-zalan-do: %v", err)
		os.Exit(1)
	}

	// TODO(linki): use watch-only?

	w, err := client.Services(api.NamespaceAll).Watch(api.ListOptions{
	// better to filter by services that have external IPs here
	// FieldSelector: fields.OneTermEqualSelector("external ip", "exists"),
	})
	if err != nil {
		fmt.Printf("Unable to watch list of services: %v", err)
		os.Exit(1)
	}
	events := w.ResultChan()

	for {
		event, ok := <-events
		if !ok {
			fmt.Println("Channel was closed")
			os.Exit(1)
		}

		// spew.Dump(event)

		if event.Type == watch.Added || event.Type == watch.Modified {
			svc := event.Object.(*api.Service)
			// spew.Dump(*svc)

			fmt.Printf("%s: %s/%s\n", event.Type, svc.Namespace, svc.Name)

			if len(svc.Status.LoadBalancer.Ingress) > 1 {
				fmt.Print("more than one ingress not supported")
				os.Exit(1)
			}

			if len(svc.Status.LoadBalancer.Ingress) == 1 {
				change := &dns.Change{
					Additions: []*dns.ResourceRecordSet{
						&dns.ResourceRecordSet{
							Name:    fmt.Sprintf("%s-%s.oolong.gcp.zalan.do.", svc.Namespace, svc.Name),
							Rrdatas: []string{svc.Status.LoadBalancer.Ingress[0].IP},
							Ttl:     300,
							Type:    "A",
						},
					},
				}

				// spew.Dump(change)

				_, err2 := dnsService.Changes.Create("zalando-teapot", "gcp-zalan-do", change).Do()
				if err2 != nil {
					if !strings.Contains(err2.Error(), "alreadyExists") {
						fmt.Printf("Unable to create change for zalando-teapot/gcp-zalan-do: %v", err2)
						os.Exit(1)
					}
				}

				time.Sleep(time.Second)
			}
		}

		if event.Type == watch.Deleted {
			svc := event.Object.(*api.Service)
			// spew.Dump(*svc)

			fmt.Printf("%s: %s/%s\n", event.Type, svc.Namespace, svc.Name)

			if len(svc.Status.LoadBalancer.Ingress) > 1 {
				fmt.Print("more than one ingress not supported")
				os.Exit(1)
			}

			if len(svc.Status.LoadBalancer.Ingress) == 1 {
				change := &dns.Change{
					Deletions: []*dns.ResourceRecordSet{
						&dns.ResourceRecordSet{
							Name:    fmt.Sprintf("%s-%s.oolong.gcp.zalan.do.", svc.Namespace, svc.Name),
							Rrdatas: []string{svc.Status.LoadBalancer.Ingress[0].IP},
							Ttl:     300,
							Type:    "A",
						},
					},
				}

				spew.Dump(change)

				_, err2 := dnsService.Changes.Create("zalando-teapot", "gcp-zalan-do", change).Do()
				if err2 != nil {
					if !strings.Contains(err2.Error(), "notFound") {
						fmt.Printf("Unable to create change for zalando-teapot/gcp-zalan-do: %v", err2)
						os.Exit(1)
					}
				}

				time.Sleep(time.Second)
			}
		}

		if event.Type == watch.Error {
			spew.Dump(event)
			fmt.Print("watch got an error event")
			os.Exit(1)
		}
	}
}
