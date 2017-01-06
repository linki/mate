package test

import (
	"fmt"

	"sync"

	"github.com/aws/aws-sdk-go/service/route53"
)

type Options struct {
	GroupID string
}

type Client struct {
	Current        map[string][]*route53.ResourceRecordSet
	LastUpsert     map[string][]*route53.ResourceRecordSet
	LastDelete     map[string][]*route53.ResourceRecordSet
	LastCreate     map[string][]*route53.ResourceRecordSet
	failNext       error
	Options        Options
	UpdateMapMutex sync.Mutex
}

func NewClient(groupID string) *Client {
	return &Client{
		Options: Options{
			GroupID: groupID,
		},
		LastCreate: map[string][]*route53.ResourceRecordSet{},
		LastDelete: map[string][]*route53.ResourceRecordSet{},
		LastUpsert: map[string][]*route53.ResourceRecordSet{},
	}
}

func (c *Client) GetGroupID() string {
	return fmt.Sprintf("\"mate:%s\"", c.Options.GroupID)
}

func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
	c.Current = getOriginalState(c.GetGroupID())
	return c.Current[zoneID], nil
}

func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error {
	c.UpdateMapMutex.Lock()
	defer c.UpdateMapMutex.Unlock()
	if len(create) > 0 {
		c.LastCreate[zoneID] = create
	}
	if len(del) > 0 {
		c.LastDelete[zoneID] = del
	}
	if len(upsert) > 0 {
		c.LastUpsert[zoneID] = upsert
	}
	return nil
}

func (c *Client) GetCanonicalZoneIDs(lbDNS []string) (map[string]string, error) {
	loadBalancersMap := map[string]string{} //map LB Dns to its canonical hosted zone id

	for _, dns := range lbDNS {
		loadBalancersMap[dns] = "random-zone-id"
	}
	return loadBalancersMap, nil
}

func (c *Client) GetHostedZones() (map[string]string, error) {
	return getHostedZones(), nil
}
