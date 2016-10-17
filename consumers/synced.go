package consumers

import (
	"sync"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type synced struct {
	sync.Mutex
	Consumer
}

// Provides a consumer that can execute only
// one operation at a time, and blocks
// concurrent operations until the current one
// finishes.
func syncedConsumer(c Consumer) Consumer {
	return &synced{Consumer: c}
}

func (s *synced) Sync(endpoints []*pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Sync(endpoints)
}

func (s *synced) Process(endpoint *pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Process(endpoint)
}
