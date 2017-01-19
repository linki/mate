package consumers

import (
	"fmt"
	"sync"

	"github.com/zalando-incubator/mate/config"
	"github.com/zalando-incubator/mate/pkg"
)

// var params struct {
// 	domain        string
// 	project       string
// 	zone          string
// 	recordGroupID string
// 	awsAccountID  string
// 	awsRole       string
// 	awsHostedZone string
// 	awsGroupID    string
// }

type Consumer interface {
	Sync([]*pkg.Endpoint) error
	Consume(<-chan *pkg.Endpoint, chan<- error, <-chan struct{}, *sync.WaitGroup)
	Process(*pkg.Endpoint) error
}

// Returns a Consumer implementation.
func New(name string, cfg *config.MateConfig) (Consumer, error) {
	switch name {
	case "google":
		return NewGoogleDNS(cfg.GoogleConfig)
	case "aws":
		return NewAWSConsumer(cfg.AWSConfig)
	case "stdout":
		return NewStdout()
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", name)
	}
}
