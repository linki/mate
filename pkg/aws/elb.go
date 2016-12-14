package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/zalando-incubator/mate/pkg"
)

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

	loadBalancerMap := map[string]string{} //map LB Dns to its canonical hosted zone id

	err = c.writeALBsToMap(loadBalancerMap, endpoints, session)
	if err != nil {
		return nil, err
	}

	err = c.writeELBsToMap(loadBalancerMap, endpoints, session)
	if err != nil {
		return nil, err
	}

	return loadBalancerMap, nil
}

func (c *Client) writeELBsToMap(loadBalancerMap map[string]string, endpoints []*pkg.Endpoint, session *session.Session) error {
	client := elb.New(session)

	resp, err := client.DescribeLoadBalancers(nil)
	if err != nil {
		return err
	}

	loadBalancers := resp.LoadBalancerDescriptions

	for _, loadbalancer := range loadBalancers {
		for _, endpoint := range endpoints {
			if aws.StringValue(loadbalancer.DNSName) == endpoint.Hostname {
				loadBalancerMap[endpoint.Hostname] = aws.StringValue(loadbalancer.CanonicalHostedZoneNameID)
				break
			}
		}
	}
	return nil
}

func (c *Client) writeALBsToMap(loadBalancerMap map[string]string, endpoints []*pkg.Endpoint, session *session.Session) error {
	client := elbv2.New(session)

	resp, err := client.DescribeLoadBalancers(nil)
	if err != nil {
		return err
	}

	loadBalancers := resp.LoadBalancers

	for _, loadbalancer := range loadBalancers {
		for _, endpoint := range endpoints {
			if aws.StringValue(loadbalancer.DNSName) == endpoint.Hostname {
				loadBalancerMap[endpoint.Hostname] = aws.StringValue(loadbalancer.CanonicalHostedZoneId)
				break
			}
		}
	}
	return nil
}
