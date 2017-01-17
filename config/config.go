package config

import kingpin "gopkg.in/alecthomas/kingpin.v2"
import "net/url"

const (

	// fake producer default config
	fakeDefaultDomain = "example.org."
	fakeIPMode        = "ip"
	fakeHostnameMode  = "hostname"
	fakeFixedMode     = "fixed"
)

type MateConfig struct {
	Producer string
	Consumer string
	Debug    bool
	SyncOnly bool

	FakeProducerConfig
	KubernetesConfig
}

type FakeProducerConfig struct {
	FakeDNSName       string
	FakeMode          string
	FakeTargetDomain  string
	FakeFixedDNSName  string
	FakeFixedIP       string
	FakeFixedHostname string
}

type KubernetesConfig struct {
	KubeServer     *url.URL
	KubeFormat     string
	TrackNodePorts bool
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

	kingpin.Flag("fake-dnsname", "The fake DNS name to use.").Default(fakeDefaultDomain).StringVar(&cfg.FakeDNSName)
	kingpin.Flag("fake-mode", "The mode to run in.").Default(fakeIPMode).StringVar(&cfg.FakeMode)
	kingpin.Flag("fake-target-domain", "The target domain for hostname mode.").Default(fakeDefaultDomain).StringVar(&cfg.FakeTargetDomain)
	kingpin.Flag("fake-fixed-dnsname", "The full fake DNS name to use.").StringVar(&cfg.FakeFixedDNSName)
	kingpin.Flag("fake-fixed-ip", "The full fake IP to use.").StringVar(&cfg.FakeFixedIP)
	kingpin.Flag("fake-fixed-hostname", "The full fake host name to use.").StringVar(&cfg.FakeFixedHostname)

	kingpin.Flag("kubernetes-server", "The address of the Kubernetes API server.").URLVar(&cfg.KubeServer)
	kingpin.Flag("kubernetes-format", "Format of DNS entries, e.g. {{.Name}}-{{.Namespace}}.example.com").StringVar(&cfg.KubeFormat)
	kingpin.Flag("kubernetes-track-node-ports", "When true, generates DNS entries for type=NodePort services").BoolVar(&cfg.TrackNodePorts)

	kingpin.Parse()
}

func (cfg *MateConfig) Validate() error {
	return nil
}
