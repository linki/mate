package consumers

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/zalando-incubator/mate/pkg"

	log "github.com/Sirupsen/logrus"
	"github.com/zalando-incubator/mate/config"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
)

const (
	heritageLabel = "heritage=mate"
	labelPrefix   = "mate/record-group-id="
)

type googleDNSConsumer struct {
	client  *dns.Service
	labels  []string
	groupID string
	project string
	zone    string
}

type ownedRecord struct {
	owner  *dns.ResourceRecordSet
	record *dns.ResourceRecordSet
}

func NewGoogleDNS(cfg *config.GoogleConfig) (Consumer, error) {
	if cfg.GoogleZone == "" {
		return nil, errors.New("Please provide --google-zone")
	}

	if cfg.GoogleProject == "" {
		return nil, errors.New("Please provide --google-project")
	}

	if cfg.GoogleRecordGroupID == "" {
		return nil, errors.New("Please provide --google-record-group-id")
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

	labels := []string{heritageLabel, labelPrefix + cfg.GoogleRecordGroupID}

	return &googleDNSConsumer{
		client:  client,
		labels:  labels,
		groupID: cfg.GoogleRecordGroupID,
		project: cfg.GoogleProject,
		zone:    cfg.GoogleZone,
	}, nil
}

func (d *googleDNSConsumer) Sync(endpoints []*pkg.Endpoint) error {
	currentRecords, err := d.currentRecords()
	if err != nil {
		return err
	}

	log.Debugln("Current records:")
	d.printRecords(currentRecords)

	change := new(dns.Change)

	records := make(map[string][]string)

	for _, e := range endpoints {
		record, exists := currentRecords[e.DNSName]

		if !exists || exists && d.isResponsible(record.owner) {
			records[e.DNSName] = append(records[e.DNSName], e.IP)
		}
	}

	for dnsName, ips := range records {
		change.Additions = append(change.Additions,
			&dns.ResourceRecordSet{
				Name:    dnsName,
				Rrdatas: ips,
				Ttl:     300,
				Type:    "A",
			},
			&dns.ResourceRecordSet{
				Name:    dnsName,
				Rrdatas: d.labels,
				Ttl:     300,
				Type:    "TXT",
			},
		)
	}

	for _, r := range currentRecords {
		if r.record != nil && d.isResponsible(r.owner) {
			change.Deletions = append(change.Deletions,
				&dns.ResourceRecordSet{
					Name:    r.record.Name,
					Rrdatas: r.record.Rrdatas,
					Ttl:     r.record.Ttl,
					Type:    r.record.Type,
				},
				&dns.ResourceRecordSet{
					Name:    r.record.Name,
					Rrdatas: d.labels,
					Ttl:     r.record.Ttl,
					Type:    "TXT",
				},
			)
		}
	}

	err = d.applyChange(change)
	if err != nil {
		return fmt.Errorf("Error applying change for %s/%s: %v", d.project, d.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) Consume(endpoints <-chan *pkg.Endpoint, errors chan<- error, done <-chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	log.Infoln("[Google] Listening for events...")

	for {
		select {
		case e, ok := <-endpoints:
			if !ok {
				log.Info("[Google] channel closed")
				return
			}

			log.Infof("[Google] Processing (%s, %s, %s)\n", e.DNSName, e.IP, e.Hostname)

			err := d.Process(e)
			if err != nil {
				errors <- err
			}
		case <-done:
			log.Info("[Google] Exited consuming loop.")
			return
		}
	}
}

func (d *googleDNSConsumer) Process(endpoint *pkg.Endpoint) error {
	change := new(dns.Change)

	change.Additions = []*dns.ResourceRecordSet{
		{
			Name:    endpoint.DNSName,
			Rrdatas: []string{endpoint.IP},
			Ttl:     300,
			Type:    "A",
		},
		&dns.ResourceRecordSet{
			Name:    endpoint.DNSName,
			Rrdatas: d.labels,
			Ttl:     300,
			Type:    "TXT",
		},
	}

	err := d.applyChange(change)
	if err != nil {
		return fmt.Errorf("Error applying change for %s/%s: %v", d.project, d.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) applyChange(change *dns.Change) error {
	if len(change.Additions) == 0 && len(change.Deletions) == 0 {
		log.Infof("Didn't submit change (no changes)")
		return nil
	}

	_, err := d.client.Changes.Create(d.project, d.zone, change).Do()
	if err != nil {
		if strings.Contains(err.Error(), "alreadyExists") {
			log.Warnf("Cannot update some DNS records (already exist)")
			return nil
		}
		return fmt.Errorf("Unable to create change for %s/%s: %v", d.project, d.zone, err)
	}

	return nil
}

func (d *googleDNSConsumer) currentRecords() (map[string]*ownedRecord, error) {
	resp, err := d.client.ResourceRecordSets.List(d.project, d.zone).Do()
	if err != nil {
		return nil, fmt.Errorf("Error getting DNS records from %s/%s: %v", d.project, d.zone, err)
	}

	records := make(map[string]*ownedRecord)

	for _, r := range resp.Rrsets {
		if r.Type == "A" || r.Type == "TXT" {
			record, exists := records[r.Name]

			if !exists {
				record = &ownedRecord{}
			}

			switch r.Type {
			case "A":
				record.record = r
			case "TXT":
				record.owner = r
			}

			records[r.Name] = record
		}
	}

	return records, nil
}

func (d *googleDNSConsumer) printRecords(records map[string]*ownedRecord) {
	for _, r := range records {
		if r.record != nil && d.isResponsible(r.owner) {
			log.Debugln(" ", r.record.Name, r.record.Type, r.record.Rrdatas)
		}
	}
}

func (d *googleDNSConsumer) isResponsible(record *dns.ResourceRecordSet) bool {
	return record != nil && d.labelsMatch(record.Rrdatas)
}

func (d *googleDNSConsumer) labelsMatch(labels []string) bool {
	return len(labels) == 2 &&
		strings.Trim(labels[0], `"`) == heritageLabel &&
		strings.Trim(labels[1], `"`) == labelPrefix+d.groupID
}
