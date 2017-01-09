package aws

import (
	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
)

//LoadBalancer struct to aggregate ELB and ALB with extracted DNSName and its canonical hosted zone id
type LoadBalancer struct {
	DNSName         string
	CanonicalZoneID string
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
