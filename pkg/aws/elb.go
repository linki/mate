package aws

import (
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/zalando-incubator/mate/pkg"
)

//LoadBalancer struct to aggregate ELB and ALB with extracted DNSName and its canonical hosted zone id
type LoadBalancer struct {
	DNSName         string
	CanonicalZoneID string
}

//GetLoadBalancerFunc is a func type to represent the interface of functions that retrieve the list
//of load balancers from AWS
type GetLoadBalancerFunc func(*session.Session) ([]*LoadBalancer, error)

//getCanonicalZoneIDs returns the map of LB (ALB + ELB classic) mapped to its CanonicalHostedZoneId
func (c *Client) getCanonicalZoneIDs(endpoints []*pkg.Endpoint) (map[string]string, error) {
	session, err := session.NewSessionWithOptions(session.Options{
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

	for _, getLBs := range []GetLoadBalancerFunc{c.getALBs, c.getELBs} {
		wg.Add(1)
		go func(getLBs GetLoadBalancerFunc) {
			defer wg.Done()
			lbs, err := getLBs(session)
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

	for _, endpoint := range endpoints {
		for _, loadBalancer := range loadBalancers {
			if endpoint.Hostname == loadBalancer.DNSName {
				loadBalancersMap[endpoint.Hostname] = loadBalancer.CanonicalZoneID
			}
		}
	}

	return loadBalancersMap, nil
}

func (c *Client) getELBs(session *session.Session) ([]*LoadBalancer, error) {
	result := make([]*LoadBalancer, 0)

	client := elb.New(session)

	params := &elb.DescribeLoadBalancersInput{}

	err := client.DescribeLoadBalancersPages(params, func(resp *elb.DescribeLoadBalancersOutput, lastPage bool) bool {
		loadBalancers := resp.LoadBalancerDescriptions
		log.Debugf("Getting a page of ELBs of length: %d", len(resp.LoadBalancerDescriptions))
		for _, loadbalancer := range loadBalancers {
			result = append(result, &LoadBalancer{
				DNSName:         aws.StringValue(loadbalancer.DNSName),
				CanonicalZoneID: aws.StringValue(loadbalancer.CanonicalHostedZoneNameID),
			})
		}
		return !lastPage
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) getALBs(session *session.Session) ([]*LoadBalancer, error) {
	result := make([]*LoadBalancer, 0)

	client := elbv2.New(session)

	params := &elbv2.DescribeLoadBalancersInput{}

	err := client.DescribeLoadBalancersPages(params, func(resp *elbv2.DescribeLoadBalancersOutput, lastPage bool) bool {
		loadBalancers := resp.LoadBalancers
		log.Debugf("Getting a page of ALBs of length: %d", len(resp.LoadBalancers))
		for _, loadbalancer := range loadBalancers {
			result = append(result, &LoadBalancer{
				DNSName:         aws.StringValue(loadbalancer.DNSName),
				CanonicalZoneID: aws.StringValue(loadbalancer.CanonicalHostedZoneId),
			})
		}
		return !lastPage
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}
