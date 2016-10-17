package awsclienttest

import (
	"errors"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type Client struct {
	HostedZone string
	Zones      map[string]map[string]string
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	z := c.Zones[c.HostedZone]
	if z == nil {
		return errors.New("test hosted zone not found")
	}

	for _, ep := range del {
		delete(z, ep.DNSName)
	}

	for _, ep := range upsert {
		z[ep.DNSName] = ep.IP
	}

	return nil
}
