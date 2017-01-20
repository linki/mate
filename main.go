package main

import (
	"fmt"

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

	p, err := newProducer(cfg)
	if err != nil {
		log.Fatalf("Error creating producer: %v", err)
	}

	c, err := newSyncConsumer(cfg)
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

func newSyncConsumer(cfg *config.MateConfig) (consumers.Consumer, error) {
	// New returns a Consumer implementation.
	var consumer consumers.Consumer
	var err error
	switch cfg.Consumer {
	case "google":
		consumer, err = consumers.NewGoogleDNS(cfg.GoogleZone, cfg.GoogleProject, cfg.GoogleRecordGroupID)
	case "aws":
		consumer, err = consumers.NewAWSConsumer(cfg.AWSRecordGroupID)
	case "stdout":
		consumer, err = consumers.NewStdout()
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", cfg.Consumer)
	}
	if err != nil {
		return nil, err
	}
	return consumers.NewSynced(consumer)
}

func newProducer(cfg *config.MateConfig) (producers.Producer, error) {
	switch cfg.Producer {
	case "kubernetes":
		kubeConfig := &producers.KubernetesOptions{
			Format:    cfg.KubeFormat,
			APIServer: cfg.KubeServer,
		}
		return producers.NewKubernetes(kubeConfig)
	case "fake":
		fakeConfig := &producers.FakeProducerOptions{
			DNSName:       cfg.FakeDNSName,
			FixedDNSName:  cfg.FakeFixedDNSName,
			FixedHostname: cfg.FakeFixedHostname,
			FixedIP:       cfg.FakeFixedIP,
			Mode:          cfg.FakeMode,
			TargetDomain:  cfg.FakeTargetDomain,
		}
		return producers.NewFake(fakeConfig)
	case "null":
		return producers.NewNull()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", cfg.Producer)
}
