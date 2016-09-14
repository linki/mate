package consumers

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.bus.zalan.do/teapot/mate/pkg"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"gopkg.in/alecthomas/kingpin.v2"
)

type googleDNSConsumer struct {
	client *dns.Service
	sync.Mutex
}

func init() {
	//	kingpin.Flag("gcloud-domain", "The DNS domain to create DNS entries under.").StringVar(params.domain)
	kingpin.Flag("gcloud-project", "Project ID that manages the zone").StringVar(&params.project)
	kingpin.Flag("gcloud-zone", "Name of the zone to manage.").StringVar(&params.zone)
}

func NewGoogleDNS() (*googleDNSConsumer, error) {
	if params.zone == "" {
		return nil, errors.New("Please provide --gcloud-zone")
	}

	if params.project == "" {
		return nil, errors.New("Please provide --gcloud-project")
	}

	// if params.zone == "" {
	// 	params.zone = strings.Replace(params.domain, ".", "-", -1)
	// 	log.Infof("No --zone provided, inferring from domain: '%s'.", params.zone)
	// }

	gcloud, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		return nil, fmt.Errorf("Error creating default client: %v", err)
	}

	client, err := dns.New(gcloud)
	if err != nil {
		return nil, fmt.Errorf("Error creating DNS service: %v", err)
	}

	return &googleDNSConsumer{
		client: client,
	}, nil
}

func (d *googleDNSConsumer) Sync(endpoints []*pkg.Endpoint) error {
	d.Lock()
	defer d.Unlock()

	records, err := d.currentRecords()
	if err != nil {
		return err
	}

	log.Debugln("Current records:")
	d.printRecords(records)

	change := new(dns.Change)

	for _, svc := range endpoints {
		change.Additions = append(change.Additions, &dns.ResourceRecordSet{
			Name:    svc.DNSName,
			Rrdatas: []string{svc.IP},
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

	err = d.applyChange(change)
	if err != nil {
		return fmt.Errorf("Error applying change for %s/%s: %v", params.project, params.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) Process(endpoint *pkg.Endpoint) error {
	d.Lock()
	defer d.Unlock()

	change := new(dns.Change)

	change.Additions = []*dns.ResourceRecordSet{
		&dns.ResourceRecordSet{
			Name:    endpoint.DNSName,
			Rrdatas: []string{endpoint.IP},
			Ttl:     300,
			Type:    "A",
		},
	}

	err := d.applyChange(change)
	if err != nil {
		return fmt.Errorf("Error applying change for %s/%s: %v", params.project, params.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) applyChange(change *dns.Change) error {
	_, err := d.client.Changes.Create(params.project, params.zone, change).Do()
	if err != nil {
		if strings.Contains(err.Error(), "alreadyExists") {
			log.Warnf("Cannot update some DNS records (already exist)")
			return nil
		}
		return fmt.Errorf("Unable to create change for %s/%s: %v", params.project, params.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) currentRecords() ([]*dns.ResourceRecordSet, error) {
	resp, err := d.client.ResourceRecordSets.List(params.project, params.zone).Do()
	if err != nil {
		return nil, fmt.Errorf("Error getting DNS records from %s/%s: %v", params.project, params.zone, err)
	}

	records := make([]*dns.ResourceRecordSet, 0)
	for _, r := range resp.Rrsets {
		if r.Type == "A" {
			records = append(records, r)
		}
	}

	return records, nil
}

func (d *googleDNSConsumer) printRecords(records []*dns.ResourceRecordSet) {
	for _, r := range records {
		log.Debugln(" ", r.Name, r.Type, r.Rrdatas)
	}
}
