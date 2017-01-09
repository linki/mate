package aws

import (
	"errors"
	"sync"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

const (
	defaultSessionDuration = 30 * time.Minute
)

// TODO: move to somewhere
type Logger interface {
	Infoln(...interface{})
}

type defaultLog struct{}

func (l defaultLog) Infoln(args ...interface{}) {
	log.Infoln(args...)
}

type Options struct {
	Log Logger
}

type Client struct {
	options Options
}

var ErrInvalidAWSResponse = errors.New("invalid AWS response")

func New(o Options) *Client {

	if o.Log == nil {
		o.Log = defaultLog{}
	}

	return &Client{o}
}

//ListRecordSets retrieve all records existing in the specified hosted zone
func (c *Client) ListRecordSets(zoneID string) ([]*route53.ResourceRecordSet, error) {
	records := make([]*route53.ResourceRecordSet, 0)

	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}

	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(zoneID),
	}

	err = client.ListResourceRecordSetsPages(params, func(resp *route53.ListResourceRecordSetsOutput, lastPage bool) bool {
		log.Debugf("Getting a list of AWS RRS of length: %d", len(resp.ResourceRecordSets))
		records = append(records, resp.ResourceRecordSets...)
		return !lastPage
	})

	if err != nil {
		return nil, err
	}

	return records, nil
}

//ChangeRecordSets creates and submits the record set change against the AWS API
func (c *Client) ChangeRecordSets(upsert, del, create []*route53.ResourceRecordSet, zoneID string) error {
	client, err := c.initRoute53Client()
	if err != nil {
		return err
	}

	var changes []*route53.Change
	changes = append(changes, createChangesList("CREATE", create)...)
	changes = append(changes, createChangesList("UPSERT", upsert)...)
	changes = append(changes, createChangesList("DELETE", del)...)
	if len(changes) > 0 {
		params := &route53.ChangeResourceRecordSetsInput{
			ChangeBatch: &route53.ChangeBatch{
				Changes: changes,
			},
			HostedZoneId: aws.String(zoneID),
		}
		_, err = client.ChangeResourceRecordSets(params)
		return err
	}
	return nil
}

// GetHostedZones returns the map hosted zone domain name -> zone id
func (c *Client) GetHostedZones() (map[string]string, error) {
	client, err := c.initRoute53Client()
	if err != nil {
		return nil, err
	}

	output, err := client.ListHostedZones(nil)
	if err != nil {
		return nil, err
	}

	hostedZoneMap := map[string]string{}
	for _, zone := range output.HostedZones {
		hostedZoneMap[aws.StringValue(zone.Name)] = aws.StringValue(zone.Id)
	}

	return hostedZoneMap, nil
}

//GetCanonicalZoneIDs returns the map of LB (ALB + ELB classic) mapped to its CanonicalHostedZoneId
func (c *Client) GetCanonicalZoneIDs(lbDNS []string) (map[string]string, error) {
	var GetLoadBalancerFunc = []func(*session.Session) ([]*LoadBalancer, error){c.getALBs, c.getELBs}

	lbSession, err := session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Logger: aws.LoggerFunc(c.options.Log.Infoln),
			CredentialsChainVerboseErrors: aws.Bool(true),
		},
	})
	if err != nil {
		return nil, err
	}

	loadBalancers := make([]*LoadBalancer, 0)

	var addLBMutex sync.Mutex
	var wg sync.WaitGroup

	for _, getLBs := range GetLoadBalancerFunc {
		wg.Add(1)
		go func(getLBs func(*session.Session) ([]*LoadBalancer, error)) {
			defer wg.Done()
			lbs, err := getLBs(lbSession)
			if err != nil {
				log.Errorf("Error getting LBs: %v. Skipping...", err)
				return
			}
			addLBMutex.Lock()
			loadBalancers = append(loadBalancers, lbs...)
			addLBMutex.Unlock()
		}(getLBs)
	}

	wg.Wait()

	loadBalancersMap := map[string]string{} //map LB Dns to its canonical hosted zone id

	for _, dns := range lbDNS {
		for _, loadBalancer := range loadBalancers {
			if dns == loadBalancer.DNSName {
				loadBalancersMap[dns] = loadBalancer.CanonicalZoneID
			}
		}
	}
	return loadBalancersMap, nil
}
