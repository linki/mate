package main

import (
	"fmt"

	log "github.com/Sirupsen/logrus"

	"github.com/zalando-incubator/mate/consumers"
	"github.com/zalando-incubator/mate/controller"
	"github.com/zalando-incubator/mate/producers"
)

var version = "Unknown"

func main() {
	cfg := newConfig(version)

	cfg.parseFlags()

	if cfg.debug {
		log.SetLevel(log.DebugLevel)
	}

	p, err := newProducer(cfg)
	if err != nil {
		log.Fatalf("Error creating producer: %v", err)
	}

	c, err := newSynchronizedConsumer(cfg)
	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}

	opts := &controller.Options{
		SyncOnly: cfg.syncOnly,
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

func newSynchronizedConsumer(cfg *mateConfig) (consumers.Consumer, error) {
	var consumer consumers.Consumer
	var err error
	switch cfg.consumer {
	case "google":
		consumer, err = consumers.NewGoogleCloudDNSConsumer(cfg.googleProject, cfg.googleRecordGroupID)
	case "aws":
		consumer, err = consumers.NewAWSRoute53Consumer(cfg.awsRecordGroupID)
	case "stdout":
		consumer, err = consumers.NewStdoutConsumer()
	default:
		return nil, fmt.Errorf("Unknown consumer '%s'.", cfg.consumer)
	}
	if err != nil {
		return nil, err
	}
	return consumers.NewSynchronizedConsumer(consumer)
}

func newProducer(cfg *mateConfig) (producers.Producer, error) {
	switch cfg.producer {
	case "kubernetes":
		kubeConfig := &producers.KubernetesOptions{
			Format:         cfg.kubernetesFormat,
			APIServer:      cfg.kubernetesServer,
			TrackNodePorts: cfg.kubernetesTrackNodePorts,
		}
		return producers.NewKubernetesProducer(kubeConfig)
	case "fake":
		fakeConfig := &producers.FakeProducerOptions{
			DNSName:       cfg.fakeDNSName,
			FixedDNSName:  cfg.fakeFixedDNSName,
			FixedHostname: cfg.fakeFixedHostname,
			FixedIP:       cfg.fakeFixedIP,
			Mode:          cfg.fakeMode,
			TargetDomain:  cfg.fakeTargetDomain,
		}
		return producers.NewFakeProducer(fakeConfig)
	case "null":
		return producers.NewNullProducer()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", cfg.producer)
}
