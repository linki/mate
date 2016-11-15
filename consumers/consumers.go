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
	Sync([]*pkg.Endpoint, string) error
	Process(*pkg.Endpoint) error
}

// Returns a Consumer implementation.
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
		return nil, fmt.Errorf("error creating consumer '%s': %v", name, err)
	}

	return c, nil
}
