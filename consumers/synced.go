package consumers

import (
	"sync"

	"github.com/zalando-incubator/mate/pkg"
)

type SynchronizedConsumer struct {
	sync.Mutex
	Consumer
}

// NewSynchronizedConsumer provides a consumer that can execute only
// one operation at a time, and blocks
// concurrent operations until the current one
// finishes.
func NewSynchronizedConsumer(consumer Consumer) (Consumer, error) {
	return &SynchronizedConsumer{Consumer: consumer}, nil
}

func (s *SynchronizedConsumer) Sync(endpoints []*pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Sync(endpoints)
}

func (s *SynchronizedConsumer) Process(endpoint *pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Process(endpoint)
}
