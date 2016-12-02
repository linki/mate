package producers

import (
	"fmt"

	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/producers/kubernetes"
)

var params struct {
	dnsName      string
	mode         string
	targetDomain string
}

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	StartWatch() error
	ResultChan() (chan *pkg.Endpoint, error)
}

func New(name string) (Producer, error) {
	switch name {
	case "kubernetes":
		return kubernetes.NewProducer()
	case "fake":
		return NewFake()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
