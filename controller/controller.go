package controller

import (
	"errors"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"

	"github.bus.zalan.do/teapot/mate/consumers"
	"github.bus.zalan.do/teapot/mate/pkg"
	"github.bus.zalan.do/teapot/mate/producers"
)

const (
	defaultSyncPeriod = 1 * time.Minute
	clusterName = "dummy"
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

func (c *Controller) Synchronize() error {
	c.wg.Add(1)

	go func() {
		for {
			log.Infoln("Synchronizing DNS entries...")

			endpoints, err := c.producer.Endpoints()
			if err != nil {
				log.Fatalf("Error getting endpoints from producer: %v", err)
			}

			err = c.consumer.Sync(endpoints, clusterName)
			if err != nil {
				log.Fatalf("Error consuming endpoints: %v", err)
			}

			time.Sleep(c.options.syncPeriod)
		}
	}()

	return nil
}

func (c *Controller) Watch() error {
	channel, err := c.monitorProducer()
	if err != nil {
		return err
	}

	err = c.processUpdates(channel)
	if err != nil {
		return err
	}

	return nil
}

func (c *Controller) Wait() {
	c.wg.Wait()
}

func (c *Controller) monitorProducer() (chan *pkg.Endpoint, error) {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			err := c.producer.StartWatch()
			switch {
			case err == pkg.ErrEventChannelClosed:
				log.Debugln("Unable to read from channel. Channel was closed. Trying to restart watch...")
			case err != nil:
				log.Fatalln(err)
			}
		}
	}()

	channel, err := c.producer.ResultChan()
	if err != nil {
		return nil, err
	}

	return channel, nil
}

func (c *Controller) processUpdates(channel chan *pkg.Endpoint) error {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		log.Infoln("Listening for events...")

		for {
			endpoint, ok := <-channel
			if !ok {
				log.Fatalln(errors.New("channel closed"))
			}

			log.Infof("Processing (%s, %s, %s)\n", endpoint.DNSName, endpoint.IP, endpoint.Hostname)

			err := c.consumer.Process(endpoint)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}()

	return nil
}
