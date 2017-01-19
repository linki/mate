package producers

import (
	"fmt"
	"sync"

	"github.com/zalando-incubator/mate/config"
	"github.com/zalando-incubator/mate/pkg"
)

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	Monitor(chan *pkg.Endpoint, chan error, chan struct{}, *sync.WaitGroup)
}

func New(cfg *config.MateConfig) (Producer, error) {
	switch cfg.Producer {
	case "kubernetes":
		kubeConfig := &KubernetesOptions{
			Format:    cfg.KubeFormat,
			APIServer: cfg.KubeServer,
		}
		return NewKubernetes(kubeConfig)
	case "fake":
		fakeConfig := &FakeProducerOptions{
			DNSName:       cfg.FakeDNSName,
			FixedDNSName:  cfg.FakeFixedDNSName,
			FixedHostname: cfg.FakeFixedHostname,
			FixedIP:       cfg.FakeFixedIP,
			Mode:          cfg.FakeMode,
			TargetDomain:  cfg.FakeTargetDomain,
		}
		return NewFake(fakeConfig)
	case "null":
		return NewNull()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", cfg.Producer)
}
