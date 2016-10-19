package consumers

import (
	"fmt"

	"github.bus.zalan.do/teapot/mate/pkg"
)

var params struct {
	domain        string
	project       string
	zone          string
	awsAccountID  string
	awsRole       string
	awsHostedZone string
	awsTTL        int
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
func New(name string) (Consumer, error) {
	var create func() (Consumer, error)
	switch name {
	case "google":
		create = NewGoogleDNS
	case "aws":
		create = NewAWSRoute53
	case "stdout":
		create = NewStdout
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", name)
	}

	c, err := create()
	if err != nil {
		return nil, fmt.Errorf("error creating Google DNS consumer: %v", err)
	}

	return syncedConsumer(c), nil
}
