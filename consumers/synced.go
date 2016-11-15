package consumers

import (
	"sync"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type SyncedConsumer struct {
	sync.Mutex
	Consumer
}

// NewSynced provides a consumer that can execute only
// one operation at a time, and blocks
// concurrent operations until the current one
// finishes.
func NewSynced(name string) (Consumer, error) {
	consumer, err := New(name)
	if err != nil {
		return nil, err
	}
	return &SyncedConsumer{Consumer: consumer}, nil
}

func (s *SyncedConsumer) Sync(endpoints []*pkg.Endpoint, clusterName string) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Sync(endpoints, clusterName)
}

func (s *SyncedConsumer) Process(endpoint *pkg.Endpoint) error {
	s.Lock()
	defer s.Unlock()
	return s.Consumer.Process(endpoint)
}
