package interfaces

import (
	"sync"

	"github.com/zalando-incubator/mate/pkg"
)

type Consumer interface {
	Sync([]*pkg.Endpoint) error
	Consume(<-chan *pkg.Endpoint, chan<- error, <-chan struct{}, *sync.WaitGroup)
	Process(*pkg.Endpoint) error
}
