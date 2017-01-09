package test

import (
	"sync"

	"github.com/aws/aws-sdk-go/service/route53"
)

type Client struct {
	HostedZones    map[string]string
	Current        map[string][]*route53.ResourceRecordSet
	LastUpsert     map[string][]*route53.ResourceRecordSet
	LastDelete     map[string][]*route53.ResourceRecordSet
	LastCreate     map[string][]*route53.ResourceRecordSet
	UpdateMapMutex sync.Mutex
}

func NewClient(groupID string, initState map[string][]*route53.ResourceRecordSet, hostedZones map[string]string) *Client {
	return &Client{
		HostedZones: hostedZones,
		Current:     initState,
		LastCreate:  map[string][]*route53.ResourceRecordSet{},
		LastDelete:  map[string][]*route53.ResourceRecordSet{},
		LastUpsert:  map[string][]*route53.ResourceRecordSet{},
	}
}

func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
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
	return c.HostedZones, nil
}
