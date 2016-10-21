package awsclienttest

import (
	"errors"

	"github.bus.zalan.do/teapot/mate/pkg"
)

type Client struct {
	Records      map[string]string
	AliasRecords map[string]string
	LastUpsert   []*pkg.Endpoint
	LastDelete   []*pkg.Endpoint
	failNext     error
}

func (c *Client) ListRecordSets() ([]*pkg.Endpoint, error) {
	if err := c.checkFailNext(); err != nil {
		return nil, err
	}

	var records []*pkg.Endpoint

	for dns, ip := range c.Records {
		records = append(records, &pkg.Endpoint{DNSName: dns, IP: ip})
	}

	for dns, hostname := range c.AliasRecords {
		records = append(records, &pkg.Endpoint{DNSName: dns, Hostname: hostname})
	}

	return records, nil
}

func (c *Client) ChangeRecordSets(upsert, del []*pkg.Endpoint) error {
	if err := c.checkFailNext(); err != nil {
		return err
	}

	c.LastDelete = del
	for _, ep := range del {
		if ep.IP != "" {
			delete(c.Records, ep.DNSName)
		} else {
			delete(c.AliasRecords, ep.DNSName)
		}
	}

	c.LastUpsert = upsert
	for _, ep := range upsert {
		if ep.IP != "" {
			if c.Records == nil {
				c.Records = make(map[string]string)
			}

			c.Records[ep.DNSName] = ep.IP
		} else {
			if c.AliasRecords == nil {
				c.AliasRecords = make(map[string]string)
			}

			c.AliasRecords[ep.DNSName] = ep.Hostname
		}
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
