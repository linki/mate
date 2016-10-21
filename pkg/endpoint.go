package pkg

// Endpoint is used to pass data from the producer to the consumer.
type Endpoint struct {

	// The DNS name to be set by the consumer for the record.
	DNSName string

	// In case of A records, the IP address value of the record.
	IP string

	// The value of ALIAS record (preferrably) or the CNAME
	// record, in case the provider receives only a hostname for
	// the service.
	Hostname string
}
