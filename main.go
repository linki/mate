package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/consumers"
	"github.bus.zalan.do/teapot/mate/controller"
	"github.bus.zalan.do/teapot/mate/producers"
)

const (
	// TODO: to run synchronize form time to time
	defaultInterval = 10 * time.Minute

	// defaultDomain = "oolong.gcp.zalan.do"

)

var params struct {
	producer string
	consumer string
	interval time.Duration
	once     bool
	debug    bool
}

var version = "Unknown"

func init() {
	kingpin.Flag("producer", "The endpoints producer to use.").Required().StringVar(&params.producer)
	kingpin.Flag("consumer", "The endpoints consumer to use.").Required().StringVar(&params.consumer)
	kingpin.Flag("interval", "Interval in Duration format, e.g. 60s.").Short('i').Default(defaultInterval.String()).DurationVar(&params.interval)
	kingpin.Flag("once", "Run once and exit").BoolVar(&params.once)
	kingpin.Flag("debug", "Enable debug logging.").BoolVar(&params.debug)
}

func main() {
	kingpin.Version(version)
	kingpin.Parse()

	if params.debug {
		log.SetLevel(log.DebugLevel)
	}

	p, err := producers.New(params.producer)
	if err != nil {
		log.Fatalf("Error creating producer: %v", err)
	}

	c, err := consumers.New(params.consumer)
	if err != nil {
		log.Fatalf("Error creating consumer: %v", err)
	}

	ctrl := controller.New(p, c)

	err = ctrl.Synchronize()
	if err != nil {
		log.Fatalln(err)
	}

	err = ctrl.Watch()
	if err != nil {
		log.Fatalln(err)
	}

	ctrl.Wait()
}
