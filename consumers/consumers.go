package consumers

import (
	"fmt"
	"sync"

	"github.com/zalando-incubator/mate/config"
	"github.com/zalando-incubator/mate/pkg"
)

// Consumer interface
type Consumer interface {
	Sync([]*pkg.Endpoint) error
	Consume(<-chan *pkg.Endpoint, chan<- error, <-chan struct{}, *sync.WaitGroup)
	Process(*pkg.Endpoint) error
}

// New returns a Consumer implementation.
func New(cfg *config.MateConfig) (Consumer, error) {
	switch cfg.Consumer {
	case "google":
		return NewGoogleDNS(cfg.GoogleZone, cfg.GoogleProject, cfg.GoogleRecordGroupID)
	case "aws":
		return NewAWSConsumer(cfg.AWSRecordGroupID)
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", cfg.Consumer)
	}
}
