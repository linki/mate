package providers

import "net/url"

var params struct {
	project string
	zone    string
	domain  string

	kubeServer string
	format     string

	lushanServer string
	authURL      *url.URL
	token        string
}

type Endpoint struct {
	DNSName string
	IP      string
}

type EndpointsProvider interface {
	Endpoints() ([]*Endpoint, error)
}

type DNSProvider interface {
	Sync([]*Endpoint) error
}
