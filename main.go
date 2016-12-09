package main

import (
	"os"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/zalando-incubator/mate/consumers"
	"github.com/zalando-incubator/mate/controller"
	"github.com/zalando-incubator/mate/deploy"
	"github.com/zalando-incubator/mate/producers"
)

var params struct {
	producer string
	consumer string
	debug    bool
	syncOnly bool
	deploy   bool
}

var version = "Unknown"

func init() {
	kingpin.Flag("producer", "The endpoints producer to use.").Required().StringVar(&params.producer)
	kingpin.Flag("consumer", "The endpoints consumer to use.").Required().StringVar(&params.consumer)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&params.debug)
	kingpin.Flag("sync-only", "Disable event watcher").BoolVar(&params.syncOnly)
	kingpin.Flag("deploy", "When set this deploys mate in the targt cluster.").BoolVar(&params.deploy)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if params.debug {
		log.SetLevel(log.DebugLevel)
	}

	if params.deploy {
		err := deploy.NewManifest(version, os.Args[1:]).Deploy()
		if err != nil {
			log.Fatalf("Error deploying manifest: %v", err)
		}

		os.Exit(0)
	}

	p, err := producers.New(params.producer)
	if err != nil {
		log.Fatalf("Error creating producer: %v", err)
	}

	c, err := consumers.NewSynced(params.consumer)
	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}

	ctrl := controller.New(p, c, nil)

	err = ctrl.Synchronize()
	if err != nil {
		log.Fatalln(err)
	}

	if !params.syncOnly {
		err = ctrl.Watch()
		if err != nil {
			log.Fatalln(err)
		}
	}

	ctrl.Wait()
}
