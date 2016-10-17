package consumers

import (
	"fmt"

	"github.bus.zalan.do/teapot/mate/awsclient"
	"github.bus.zalan.do/teapot/mate/pkg"
)

var params struct {
	domain  string
	project string
	zone    string
}

// Options are used to initialize a Consumer.
type Options struct {
	Name       string
	AWSOptions awsclient.Options
}

type Consumer interface {
	Sync([]*pkg.Endpoint) error
	Process(*pkg.Endpoint) error
}

// Returns a synced Consumer implementation.
//
// TODO: consider whether the syncing is necessary,
// and just drop if not. Usually, it is better to
// care about syncing in a single place, optimally
// on the caller side.
func New(o Options) (Consumer, error) {
	var (
		c   Consumer
		err error
	)

	switch o.Name {
	case "google":
		c, err = NewGoogleDNS()
	case "aws":
		c = NewAWS(awsclient.New(o.AWSOptions))
	case "stdout":
		c, err = NewStdout()
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", o.Name)
	}

	if err != nil {
		return nil, fmt.Errorf("error creating Google DNS consumer: %v", err)
	}

	return syncedConsumer(c), nil
}
