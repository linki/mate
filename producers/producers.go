package producers

import (
	"fmt"
	"sync"

	"github.com/zalando-incubator/mate/pkg"
)

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	Monitor(chan *pkg.Endpoint, chan error, chan struct{}, *sync.WaitGroup)
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
