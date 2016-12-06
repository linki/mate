package controller

import (
	"errors"
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

	options *Options

	mutex sync.Mutex
	wg    sync.WaitGroup
}

type Options struct {
	syncPeriod time.Duration
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
	}
}

func (c *Controller) Run() chan error {
	errors := make(chan error)

	errors1 := c.Synchronize()
	errors2 := c.Watch()

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			select {
			case err := <-errors1:
				errors <- err
			case err := <-errors2:
				errors <- err
			}
		}
	}()

	return errors
}

func (c *Controller) RunAndWait() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	errors := c.Run()

	for {
		select {
		case err := <-errors:
			log.Error(err)
		case <-signalChan:
			log.Info("Shutdown signal received, exiting...")
			return
		}
	}
}

func (c *Controller) Synchronize() chan error {
	errors := make(chan error)

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			log.Infoln("[Synchronize] Synchronizing DNS entries...")

			endpoints, err := c.producer.Endpoints()
			if err != nil {
				errors <- fmt.Errorf("Error getting endpoints from producer: %v", err)
			}

			err = c.consumer.Sync(endpoints)
			if err != nil {
				errors <- fmt.Errorf("Error consuming endpoints: %v", err)
			}

			log.Infof("[Synchronize] Sleeping for %s...", c.options.syncPeriod)
			select {
			case <-time.After(c.options.syncPeriod):
			}
		}
	}()

	return errors
}

func (c *Controller) Watch() chan error {
	errors := make(chan error)

	channel, errors1 := c.monitorProducer()
	errors2 := c.consumeEndpoints(channel)

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			select {
			case err := <-errors1:
				errors <- err
			case err := <-errors2:
				errors <- err
			}
		}
	}()

	return errors
}

func (c *Controller) monitorProducer() (chan *pkg.Endpoint, chan error) {
	return c.producer.Monitor()
}

func (c *Controller) consumeEndpoints(channel chan *pkg.Endpoint) chan error {
	errChan := make(chan error)

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		log.Infoln("[Watch] Listening for events...")

		for {
			endpoint, ok := <-channel
			if !ok {
				errChan <- errors.New("[Watch] channel closed")
			}

			log.Infof("[Watch] Processing (%s, %s, %s)\n", endpoint.DNSName, endpoint.IP, endpoint.Hostname)

			err := c.consumer.Process(endpoint)
			if err != nil {
				errChan <- err
			}
		}
	}()

	return errChan
}
