package awsclient

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/zalando-incubator/mate/pkg"
)

func (c *Client) initELBClient() (*elb.ELB, error) {
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
	return elb.New(session), nil
}

func (c *Client) getELBDescriptions(eps []*pkg.Endpoint) ([]*elb.LoadBalancerDescription, error) {
	client, err := c.initELBClient()
	if err != nil {
		return nil, err
	}
	var names []*string
	for _, ep := range eps {
		elbName := extractELBName(ep.Hostname)
		names = append(names, aws.String(elbName))
	}
	params := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: names,
		// PageSize:          aws.Int64(1), use default 400 records
	}
	resp, err := client.DescribeLoadBalancers(params)
	if err != nil {
		return nil, err
	}
	return resp.LoadBalancerDescriptions, nil
}

func getELBZoneID(ep *pkg.Endpoint, elbs []*elb.LoadBalancerDescription) *string {
	for _, elb := range elbs {
		if ep.Hostname == aws.StringValue(elb.DNSName) {
			return elb.CanonicalHostedZoneNameID
		}
	}
	return nil
}

func extractELBName(dns string) string {
	dns = strings.TrimPrefix(dns, "internal-") // internal elb
	lastLvlDomain := dns[:strings.Index(dns, ".")]
	timestampIndex := strings.LastIndex(lastLvlDomain, "-")
	return dns[:timestampIndex]
}
