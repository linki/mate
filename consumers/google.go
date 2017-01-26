package consumers

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/zalando-incubator/mate/pkg"

	log "github.com/Sirupsen/logrus"
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
	zones   map[string]*dns.ManagedZone
	labels  []string
	groupID string
	project string
}

type ownedRecord struct {
	owner  *dns.ResourceRecordSet
	record *dns.ResourceRecordSet
}

// NewGoogleCloudDNSConsumer creates
func NewGoogleCloudDNSConsumer(googleProject, googleRecordGroupID string) (Consumer, error) {
	if googleProject == "" {
		return nil, errors.New("Please provide --google-project")
	}

	if googleRecordGroupID == "" {
		return nil, errors.New("Please provide --google-record-group-id")
	}

	gcloud, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		return nil, fmt.Errorf("Error creating default client: %v", err)
	}

	client, err := dns.New(gcloud)
	if err != nil {
		return nil, fmt.Errorf("Error creating DNS service: %v", err)
	}

	resp, err := client.ManagedZones.List(googleProject).Do()
	if err != nil {
		return nil, fmt.Errorf("Error getting managed zones in project %s: %v", googleProject, err)
	}

	zones := make(map[string]*dns.ManagedZone)
	for _, z := range resp.ManagedZones {
		zones[z.DnsName] = z
	}

	labels := []string{heritageLabel, labelPrefix + googleRecordGroupID}

	return &googleDNSConsumer{
		client:  client,
		zones:   zones,
		labels:  labels,
		groupID: googleRecordGroupID,
		project: googleProject,
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
		return fmt.Errorf("Error applying change for project %s: %v", d.project, err)
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
		return fmt.Errorf("Error applying change for project %s: %v", d.project, err)
	}

	return nil
}

func (d *googleDNSConsumer) applyChange(change *dns.Change) error {
	if len(change.Additions) == 0 && len(change.Deletions) == 0 {
		log.Infof("Didn't submit change (no changes)")
		return nil
	}

	additions := make(map[string][]*dns.ResourceRecordSet)
	deletions := make(map[string][]*dns.ResourceRecordSet)

	for _, add := range change.Additions {
		hostedZone := d.hostedZoneFor(add.Name)
		additions[hostedZone] = append(additions[hostedZone], add)
	}
	for _, del := range change.Deletions {
		hostedZone := d.hostedZoneFor(del.Name)
		deletions[hostedZone] = append(deletions[hostedZone], del)
	}

	changes := make(map[string]*dns.Change)
	for _, z := range d.zones {
		if len(additions[z.Name]) == 0 && len(deletions[z.Name]) == 0 {
			log.Debugf("Didn't submit change for zone %s (no changes)", z.Name)
			continue
		}

		changes[z.Name] = &dns.Change{
			Additions: additions[z.Name],
			Deletions: deletions[z.Name],
		}
	}

	for z, c := range changes {
		_, err := d.client.Changes.Create(d.project, z, c).Do()
		if err != nil {
			if strings.Contains(err.Error(), "alreadyExists") {
				log.Warnf("Cannot update some DNS records (already exist)")
				continue
			}
			log.Errorf("Unable to create change for %s/%s: %v", d.project, z, err)
		}
	}

	return nil
}

func (d *googleDNSConsumer) currentRecords() (map[string]*ownedRecord, error) {
	aggregatedRecords := make([]*dns.ResourceRecordSet, 0)

	for _, z := range d.zones {
		resp, err := d.client.ResourceRecordSets.List(d.project, z.Name).Do()
		if err != nil {
			return nil, fmt.Errorf("Error getting DNS records from %s/%s: %v", d.project, z.Name, err)
		}

		aggregatedRecords = append(aggregatedRecords, resp.Rrsets...)
	}

	records := make(map[string]*ownedRecord)

	for _, r := range aggregatedRecords {
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

func (d *googleDNSConsumer) hostedZoneFor(name string) string {
	var matchName string
	var matchID string
	for zoneName, zone := range d.zones {
		if strings.HasSuffix(name, zoneName) && len(zoneName) > len(matchName) { //get the longest match for the dns name
			matchName = zoneName
			matchID = zone.Name
		}
	}
	return matchID
}

func (d *googleDNSConsumer) isResponsible(record *dns.ResourceRecordSet) bool {
	return record != nil && d.labelsMatch(record.Rrdatas)
}

func (d *googleDNSConsumer) labelsMatch(labels []string) bool {
	return len(labels) == 2 &&
		strings.Trim(labels[0], `"`) == heritageLabel &&
		strings.Trim(labels[1], `"`) == labelPrefix+d.groupID
}
