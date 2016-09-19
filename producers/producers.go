package producers

import (
	"fmt"
	"net/url"

	"github.bus.zalan.do/teapot/mate/pkg"
)

var params struct {
	dnsName   string
	ipAddress string

	project string
	zone    string
	domain  string

	kubeServer *url.URL
	format     string

	lushanServer string
	authURL      *url.URL
	token        string
}

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	StartWatch() error
	ResultChan() (chan *pkg.Endpoint, error)
}

func New(name string) (Producer, error) {
	switch name {
	case "kubernetes":
		p, err := NewKubernetes()
		if err != nil {
			return nil, fmt.Errorf("Error creating Kubernetes producer: %v", err)
		}
		return p, nil
	case "fake":
		p, err := NewFake()
		if err != nil {
			return nil, fmt.Errorf("Error creating Fake producer: %v", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
