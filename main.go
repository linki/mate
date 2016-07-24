package main

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/providers"
)

const (
	defaultInterval = 10 * time.Minute
)

var params struct {
	endpoints string
	dns       string
	interval  time.Duration
	once      bool
	debug     bool
}

var version = "Unknown"

func init() {
	kingpin.Flag("endpoints-provider", "The endpoints provider to use.").Short('e').Required().StringVar(&params.endpoints)
	kingpin.Flag("dns-provider", "The DNS provider to use.").Short('d').Required().StringVar(&params.dns)
	kingpin.Flag("interval", "Interval in Duration format, e.g. 60s.").Short('i').Default(defaultInterval.String()).DurationVar(&params.interval)
	kingpin.Flag("once", "Run once and exit").BoolVar(&params.once)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&params.debug)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if params.debug {
		log.SetLevel(log.DebugLevel)
	}

	p, err := endpointsProvider(params.endpoints)
	if err != nil {
		log.Fatalf("Error creating endpoints provider: %v", err)
	}

	d, err := dnsProvider(params.dns)
	if err != nil {
		log.Fatalf("Error creating DNS provider: %v", err)
	}

	for {
		endpoints, err := p.Endpoints()
		if err != nil {
			log.Fatalf("Unable to get list of endpoints from endpoint provider: %v", err)
		}

		err = d.Sync(endpoints)
		if err != nil {
			log.Fatalf("Unable to sync DNS entries with DNS provider: %v", err)
		}

		if params.once {
			break
		}

		log.Printf("Sleeping for %s", params.interval)
		time.Sleep(params.interval)
	}
}

func endpointsProvider(name string) (providers.EndpointsProvider, error) {
	switch name {
	case "kubernetes":
		p, err := providers.NewKubernetesProvider()
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize Kubernetes provider: %v", err)
		}
		return p, nil
	case "lushan":
		p, err := providers.NewLushanProvider()
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize Lushan provider: %v", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("Unknown endpoints provider '%s'.", name)
}

func dnsProvider(name string) (providers.DNSProvider, error) {
	switch name {
	case "google":
		p, err := providers.NewGoogleCloudDNSProvider()
		if err != nil {
			return nil, fmt.Errorf("Unable to initialize Google Cloud DNS provider: %v", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("Unknown DNS provider '%s'.", name)
}
