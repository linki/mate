package producers

import (
	"fmt"
	"net/url"

	"github.com/zalando-incubator/mate/pkg"
)

var params struct {
	dnsName      string
	mode         string
	targetDomain string

	project string
	zone    string
	domain  string

	kubeServer *url.URL
	format     string
}

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	StartWatch() error
	ResultChan() (chan *pkg.Endpoint, error)
}

func New(name string) (Producer, error) {
	switch name {
	case "kubernetes":
		return NewKubernetes()
	case "fake":
		return NewFake()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
