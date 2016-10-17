package awsclienttest

import (
	"errors"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type Client struct {
	Records      map[string]string
	LastUpsert []*pkg.Endpoint
	LastDelete []*pkg.Endpoint
	failNext error
}

func (c *Client) ListRecordSets() ([]*pkg.Endpoint, error) {
	if err := c.checkFailNext(); err != nil {
		return nil, err
	}

	var records []*pkg.Endpoint
	for dns, ip := range c.Records {
		records = append(records, &pkg.Endpoint{DNSName: dns, IP: ip})
	}

	return records, nil
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}

	c.LastDelete = del
	for _, ep := range del {
		delete(c.Records, ep.DNSName)
	}

	c.LastUpsert = upsert
	for _, ep := range upsert {
		c.Records[ep.DNSName] = ep.IP
	}

	return nil
}

func (c *Client) FailNext() {
	c.failNext = errors.New("test error")
}

func (c *Client) checkFailNext() (err error) {
	err, c.failNext = c.failNext, nil
	return
}
