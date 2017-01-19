package config

import kingpin "gopkg.in/alecthomas/kingpin.v2"
import "net/url"

type MateConfig struct {
	Producer string
	Consumer string
	Debug    bool
	SyncOnly bool

	FakeDNSName       string
	FakeMode          string
	FakeTargetDomain  string
	FakeFixedDNSName  string
	FakeFixedIP       string
	FakeFixedHostname string

	KubeServer     *url.URL
	KubeFormat     string
	TrackNodePorts bool

	AWSRecordGroupID string

	GoogleProject       string
	GoogleZone          string
	GoogleRecordGroupID string
}

func New(version string) *MateConfig {
	kingpin.Version(version)
	return &MateConfig{}
}

func (cfg *MateConfig) ParseFlags() {
	kingpin.Flag("producer", "The endpoints producer to use.").Required().StringVar(&cfg.Producer)
	kingpin.Flag("consumer", "The endpoints consumer to use.").Required().StringVar(&cfg.Consumer)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&cfg.Debug)
	kingpin.Flag("sync-only", "Disable event watcher").BoolVar(&cfg.SyncOnly)

	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").StringVar(&cfg.FakeDNSName)
	kingpin.Flag("fake-mode", "The mode to run in.").StringVar(&cfg.FakeMode)
	kingpin.Flag("fake-target-domain", "The target domain for hostname mode.").StringVar(&cfg.FakeTargetDomain)
	kingpin.Flag("fake-fixed-dnsname", "The full fake DNS name to use.").StringVar(&cfg.FakeFixedDNSName)
	kingpin.Flag("fake-fixed-ip", "The full fake IP to use.").StringVar(&cfg.FakeFixedIP)
	kingpin.Flag("fake-fixed-hostname", "The full fake host name to use.").StringVar(&cfg.FakeFixedHostname)

	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&cfg.KubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&cfg.KubeFormat)
	kingpin.Flag("kubernetes-track-node-ports", "When true, generates DNS entries for type=NodePort services").BoolVar(&cfg.TrackNodePorts)

	kingpin.Flag("aws-record-group-id", "Identifier to filter mate created records ").StringVar(&cfg.AWSRecordGroupID)

	kingpin.Flag("google-project", "Project ID that manages the zone").StringVar(&cfg.GoogleProject)
	kingpin.Flag("google-zone", "Name of the zone to manage.").StringVar(&cfg.GoogleZone)
	kingpin.Flag("google-record-group-id", "Name of the zone to manage.").StringVar(&cfg.GoogleRecordGroupID)

	kingpin.Parse()
}

func (cfg *MateConfig) Validate() error {
	return nil
}
