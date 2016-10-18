package main

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.bus.zalan.do/teapot/mate/awsclient"
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
	producer           string
	consumer           string
	awsAccountID       string
	awsRole            string
	awsHostedZone      string
	awsTTL             int
	interval           time.Duration
	once               bool
	debug              bool
}

var version = "Unknown"

func init() {
	kingpin.Flag("producer", "The endpoints producer to use.").Required().StringVar(&params.producer)
	kingpin.Flag("consumer", "The endpoints consumer to use.").Required().StringVar(&params.consumer)
	kingpin.Flag("aws-account", "The ID of the AWS account to be used with the AWS consumer (required with AWS).").StringVar(&params.awsAccountID)
	kingpin.Flag("aws-role", "The AWS role to be used with the AWS consumer (required with AWS).").StringVar(&params.awsRole)
	kingpin.Flag("aws-hosted-zone", "The hosted zone name for the AWS consumer (required with AWS).").StringVar(&params.awsHostedZone)
	kingpin.Flag("aws-record-set-ttl", "TTL for the record sets created by the AWS consumer.").IntVar(&params.awsTTL)
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

	c, err := consumers.New(consumers.Options{
		Name: params.consumer,

		// used only in case consumer=aws
		AWSOptions: awsclient.Options{
			AccountID:       params.awsAccountID,
			Role:            params.awsRole,
			HostedZone:      params.awsHostedZone,
			RecordSetTTL:    params.awsTTL,
		},
	})
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
