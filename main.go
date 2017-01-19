package main

import (
	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/config"
	"github.com/zalando-incubator/mate/consumers"
	"github.com/zalando-incubator/mate/controller"
	"github.com/zalando-incubator/mate/producers"
)

var version = "Unknown"

func main() {
	cfg := config.New(version)

	cfg.ParseFlags()

	if cfg.Debug {
		log.SetLevel(log.DebugLevel)
	}

	p, err := producers.New(cfg)
	if err != nil {
		log.Fatalf("Error creating producer: %v", err)
	}

	c, err := consumers.NewSynced(cfg)
	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}

	opts := &controller.Options{
		SyncOnly: cfg.SyncOnly,
	}
	ctrl := controller.New(p, c, opts)
	errors := ctrl.Run()

	go func() {
		for {
			log.Error(<-errors)
		}
	}()

	ctrl.Wait()
}
