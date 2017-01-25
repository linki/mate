package main

import (
	"net/url"

	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type mateConfig struct {
	producer string
	consumer string
	debug    bool
	syncOnly bool

	fakeDNSName       string
	fakeMode          string
	fakeTargetDomain  string
	fakeFixedDNSName  string
	fakeFixedIP       string
	fakeFixedHostname string

	kubernetesServer         *url.URL
	kubernetesFormat         string
	kubernetesTrackNodePorts bool

	awsRecordGroupID string

	googleProject       string
	googleZone          string
	googleRecordGroupID string
}

func newConfig(version string) *mateConfig {
	kingpin.Version(version)
	return &mateConfig{}
}

func (cfg *mateConfig) parseFlags() {
	kingpin.Flag("producer", "The endpoints producer to use.").Required().StringVar(&cfg.producer)
	kingpin.Flag("consumer", "The endpoints consumer to use.").Required().StringVar(&cfg.consumer)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&cfg.debug)
	kingpin.Flag("sync-only", "Disable event watcher").BoolVar(&cfg.syncOnly)

	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").StringVar(&cfg.fakeDNSName)
	kingpin.Flag("fake-mode", "The mode to run in.").StringVar(&cfg.fakeMode)
	kingpin.Flag("fake-target-domain", "The target domain for hostname mode.").StringVar(&cfg.fakeTargetDomain)
	kingpin.Flag("fake-fixed-dnsname", "The full fake DNS name to use.").StringVar(&cfg.fakeFixedDNSName)
	kingpin.Flag("fake-fixed-ip", "The full fake IP to use.").StringVar(&cfg.fakeFixedIP)
	kingpin.Flag("fake-fixed-hostname", "The full fake host name to use.").StringVar(&cfg.fakeFixedHostname)

	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&cfg.kubernetesServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&cfg.kubernetesFormat)
	kingpin.Flag("kubernetes-track-node-ports", "When true, generates DNS entries for type=NodePort services").BoolVar(&cfg.kubernetesTrackNodePorts)

	kingpin.Flag("aws-record-group-id", "Identifier to filter mate created records ").StringVar(&cfg.awsRecordGroupID)

	kingpin.Flag("google-project", "Project ID that manages the zone").StringVar(&cfg.googleProject)
	kingpin.Flag("google-zone", "Name of the zone to manage.").StringVar(&cfg.googleZone)
	kingpin.Flag("google-record-group-id", "Name of the zone to manage.").StringVar(&cfg.googleRecordGroupID)

	kingpin.Parse()
}

func (cfg *mateConfig) validate() error {
	return nil
}
