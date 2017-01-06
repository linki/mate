package controller

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/consumers"
	"github.com/zalando-incubator/mate/pkg"
	"github.com/zalando-incubator/mate/producers"
)

const (
	defaultSyncPeriod = 1 * time.Minute
)

type Controller struct {
	producer producers.Producer
	consumer consumers.Consumer
	options  *Options

	// results is used to pass endpoints from producer to consumer
	results chan *pkg.Endpoint

	// errors can be used by producers and consumers to report errors up
	errors chan error

	// done is used to send termination signals to nested goroutines
	done chan struct{}

	// wg keeps track of running goroutines
	wg sync.WaitGroup
}

type Options struct {
	syncPeriod time.Duration
	SyncOnly   bool
}

func New(producer producers.Producer, consumer consumers.Consumer, options *Options) *Controller {
	if options == nil {
		options = &Options{}
	}

	if options.syncPeriod == 0 {
		options.syncPeriod = defaultSyncPeriod
	}

	return &Controller{
		producer: producer,
		consumer: consumer,
		options:  options,

		results: make(chan *pkg.Endpoint),
		errors:  make(chan error),
		done:    make(chan struct{}),
	}
}

func (c *Controller) Run() chan error {
	go c.Synchronize()

	if !c.options.SyncOnly {
		go c.Watch()
	}

	return c.errors
}

func (c *Controller) Wait() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan
	log.Info("Shutdown signal received, exiting...")
	close(c.done)
	c.wg.Wait()
}

func (c *Controller) Synchronize() {
	c.wg.Add(1)
	defer c.wg.Done()

	for {
		log.Debugf("[Synchronize] Sleeping for %s...", c.options.syncPeriod)
		select {
		case <-time.After(c.options.syncPeriod):
		case <-c.done:
			log.Info("[Synchronize] Exited synchronization loop.")
			return
		}

		log.Infoln("[Synchronize] Synchronizing DNS entries...")

		endpoints, err := c.producer.Endpoints()
		if err != nil {
			c.errors <- fmt.Errorf("[Synchronize] Error getting endpoints from producer: %v", err)
			continue
		}

		err = c.consumer.Sync(endpoints)
		if err != nil {
			c.errors <- fmt.Errorf("[Synchronize] Error consuming endpoints: %v", err)
			continue
		}
	}
}

func (c *Controller) Watch() {
	go c.monitorProducer()
	go c.consumeEndpoints()
}

func (c *Controller) monitorProducer() {
	c.producer.Monitor(c.results, c.errors, c.done, &c.wg)
}

func (c *Controller) consumeEndpoints() {
	c.consumer.Consume(c.results, c.errors, c.done, &c.wg)
}
