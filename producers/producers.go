package producers

import (
	"fmt"

	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/producers/fake"
	"github.com/zalando-incubator/mate/producers/kubernetes"
)

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
		return fake.NewFake()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
