package consumers

import (
	"sync"

	"github.com/zalando-incubator/mate/config"
	"github.com/zalando-incubator/mate/pkg"
)

type SyncedConsumer struct {
	sync.Mutex
	Consumer
}

// NewSynced provides a consumer that can execute only
// one operation at a time, and blocks
// concurrent operations until the current one
// finishes.
func NewSynced(name string, cfg *config.MateConfig) (Consumer, error) {
	consumer, err := New(name, cfg)
	if err != nil {
		return nil, err
	}
	return &SyncedConsumer{Consumer: consumer}, nil
}

func (s *SyncedConsumer) Sync(endpoints []*pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Sync(endpoints)
}

func (s *SyncedConsumer) Process(endpoint *pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Process(endpoint)
}
