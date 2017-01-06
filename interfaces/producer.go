package interfaces

import (
	"sync"

	"github.com/zalando-incubator/mate/pkg"
)

type Producer interface {
	Endpoints() ([]*pkg.Endpoint, error)
	Monitor(chan *pkg.Endpoint, chan error, chan struct{}, *sync.WaitGroup)
}
