package providers

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	defaultDomain = "oolong.gcp.zalan.do"
)

type googleCloudDNSProvider struct {
	client *dns.Service
}

// var params struct {
// 	project string
// 	zone    string
// 	domain  string
// }

func init() {
	kingpin.Flag("project", "Project ID that manages the zone").Required().StringVar(&params.project)
	kingpin.Flag("zone", "Name of the zone to manage.").StringVar(&params.zone)
	kingpin.Flag("domain", "The DNS domain by which cluster endpoints are reachable.").Default(defaultDomain).StringVar(&params.domain)
}

func (d *googleCloudDNSProvider) Sync(endpoints []*Endpoint) error {
	resp, err := d.client.ResourceRecordSets.List(params.project, params.zone).Do()
	if err != nil {
		return fmt.Errorf("Unable to retrieve resource record sets of %s/%s: %v", params.project, params.zone, err)
	}

	records := make([]*dns.ResourceRecordSet, 0, len(resp.Rrsets))

	for _, r := range resp.Rrsets {
		if r.Type == "A" && strings.Contains(r.Name, params.domain) {
			records = append(records, r)
		}
	}

	log.Debugln("Current A records and where they point to:")
	log.Debugln("==========================================")
	for _, r := range records {
		log.Debugln(r.Name, r.Type, r.Rrdatas)
	}

	change := new(dns.Change)

	for _, svc := range endpoints {
		dnsName := fmt.Sprintf("%s.%s.", svc.DNSName, params.domain)

		change.Additions = append(change.Additions, &dns.ResourceRecordSet{
			Name:    dnsName,
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

	_, err = d.client.Changes.Create(params.project, params.zone, change).Do()
	if err != nil {
		return fmt.Errorf("Unable to create change for %s/%s: %v", params.project, params.zone, err)
	}

	return nil
}

func NewGoogleCloudDNSProvider() (*googleCloudDNSProvider, error) {
	if params.zone == "" {
		params.zone = strings.Replace(params.domain, ".", "-", -1)
		log.Infof("No --zone provided, inferring from domain: '%s'.", params.zone)
	}

	dnsClient, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		return nil, fmt.Errorf("Unable to get DNS client: %v", err)
	}

	dnsService, err := dns.New(dnsClient)
	if err != nil {
		return nil, fmt.Errorf("Unable to create DNS client: %v", err)
	}

	return &googleCloudDNSProvider{
		client: dnsService,
	}, nil
}
