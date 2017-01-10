package producers

import (
	"sync"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/pkg"
)

type nullProducer struct{}

func NewNull() (*nullProducer, error) {
	return &nullProducer{}, nil
}

func (a *nullProducer) Endpoints() ([]*pkg.Endpoint, error) {
	return make([]*pkg.Endpoint, 0), nil
}

func (a *nullProducer) Monitor(results chan *pkg.Endpoint, errChan chan error, done chan struct{}, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	<-done

	log.Info("[Noop] Exited monitoring loop.")
}
