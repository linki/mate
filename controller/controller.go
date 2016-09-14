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

type Controller struct {
	producer producers.Producer
	consumer consumers.Consumer

	mutex sync.Mutex
	wg    sync.WaitGroup
}

func New(producer producers.Producer, consumer consumers.Consumer) *Controller {
	return &Controller{
		producer: producer,
		consumer: consumer,
	}
}

func (c *Controller) Synchronize() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			log.Infoln("Synchronizing DNS entries...")

			endpoints, err := c.producer.Endpoints()
			if err != nil {
				log.Fatalf("Error getting endpoints from producer: %v", err)
			}

			err = c.consumer.Sync(endpoints)
			if err != nil {
				log.Fatalf("Error consuming endpoints: %v", err)
			}

			time.Sleep(30 * time.Second)
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

		err := c.producer.StartWatch()
		if err != nil {
			log.Fatalln(err)
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

			log.Infof("Processing (%s, %s)\n", endpoint.DNSName, endpoint.IP)

			err := c.consumer.Process(endpoint)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}()

	return nil
}
