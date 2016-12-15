package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/zalando-incubator/mate/pkg"
)

//LoadBalancer struct to aggregate ELB and ALB with extracted DNSName and its canonnical hosted zone id
type LoadBalancer struct {
	DNSName     string
	CanonZoneID string
}

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

	albs, err := c.getALBs(session)
	if err != nil {
		return nil, err
	}

	elbs, err := c.getELBs(session)
	if err != nil {
		return nil, err
	}

	loadBalancers := append(albs, elbs...)
	loadBalancersMap := map[string]string{} //map LB Dns to its canonical hosted zone id

	for _, endpoint := range endpoints {
		for _, loadBalancer := range loadBalancers {
			if endpoint.Hostname == loadBalancer.DNSName {
				loadBalancersMap[endpoint.Hostname] = loadBalancer.CanonZoneID
			}
		}
	}

	return loadBalancersMap, nil
}

func (c *Client) getELBs(session *session.Session) ([]*LoadBalancer, error) {
	client := elb.New(session)

	resp, err := client.DescribeLoadBalancers(nil)
	if err != nil {
		return nil, err
	}

	loadBalancers := resp.LoadBalancerDescriptions
	result := make([]*LoadBalancer, len(loadBalancers))
	for i, loadbalancer := range loadBalancers {
		result[i] = &LoadBalancer{
			DNSName:     aws.StringValue(loadbalancer.DNSName),
			CanonZoneID: aws.StringValue(loadbalancer.CanonicalHostedZoneNameID),
		}
	}
	return result, nil
}

func (c *Client) getALBs(session *session.Session) ([]*LoadBalancer, error) {
	client := elbv2.New(session)

	resp, err := client.DescribeLoadBalancers(nil)
	if err != nil {
		return nil, err
	}

	loadBalancers := resp.LoadBalancers
	result := make([]*LoadBalancer, len(loadBalancers))
	for i, loadbalancer := range loadBalancers {
		result[i] = &LoadBalancer{
			DNSName:     aws.StringValue(loadbalancer.DNSName),
			CanonZoneID: aws.StringValue(loadbalancer.CanonicalHostedZoneId),
		}
	}
	return result, nil
}
