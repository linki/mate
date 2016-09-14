package consumers

import (
	"fmt"

	"github.bus.zalan.do/teapot/mate/pkg"
)

var params struct {
	domain  string
	project string
	zone    string
}

type Consumer interface {
	Sync([]*pkg.Endpoint) error
	Process(*pkg.Endpoint) error
}

func New(name string) (Consumer, error) {
	switch name {
	case "google":
		c, err := NewGoogleDNS()
		if err != nil {
			return nil, fmt.Errorf("Error creating Google DNS consumer: %v", err)
		}
		return c, nil
	case "stdout":
		c, err := NewStdout()
		if err != nil {
			return nil, fmt.Errorf("Error creating Stdout consumer: %v", err)
		}
		return c, nil
	}
	return nil, fmt.Errorf("Unknown consumer '%s'.", name)
}
