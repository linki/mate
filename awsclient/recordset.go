package awsclient

import (
	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/aws"
	"fmt"
)

//handle upsert and delete cases separately for ease of understanding
func (c *Client) actionRecords(action string, zoneID *string, eps []*pkg.Endpoint) []*route53.Change {
  var changes []*route53.Change
  for _, ep := range eps {
    changes = append(changes, &route53.Change{
      Action: aws.String(action),
      ResourceRecordSet: ep.AWSARecordAlias(zoneID, int64(c.options.RecordSetTTL)),
    })
    changes = append(changes, &route53.Change{
      Action: aws.String(action),
      ResourceRecordSet: ep.AWSTXTRecord(int64(c.options.RecordSetTTL)),
    })
  }
  return changes
}

//get record sets in raw format
func (c *Client) getRecordSets() ([]*route53.ResourceRecordSet, error) {
	client, err := c.initClient()
	if err != nil {
		return nil, err
	}

	zoneID, err := c.getZoneID(client)
	if err != nil {
		return nil, err
	}

	if zoneID == nil {
		return nil, fmt.Errorf("hosted zone not found: %s", c.options.HostedZone)
	}

	// TODO: implement paging
	params := &route53.ListResourceRecordSetsInput{
		HostedZoneId: zoneID,
	}

	rsp, err := client.ListResourceRecordSets(params)
	if err != nil {
		return nil, err
	}

	if rsp == nil {
		return nil, ErrInvalidAWSResponse
	}

	return rsp.ResourceRecordSets, nil
}
//filter out to include only mate created resource records
func filterMate(allrs []*route53.ResourceRecordSet, clusterName string) []*route53.ResourceRecordSet {
  matenames := make([]string, 0, 0)
  res := make([]*route53.ResourceRecordSet, 0, 0)
  for _, rs := range allrs {
    if aws.StringValue(rs.Type) == "TXT" && len(rs.ResourceRecords) == 1 {
      resource := rs.ResourceRecords[0]
      if aws.StringValue(resource.Value) == pkg.GetMateValue(clusterName) {
        matenames = append(matenames, *rs.Name)
      }
    }
  }
  for _, rs := range allrs {
    if aws.StringValue(rs.Type) != "A"{
      continue
    }
    name := aws.StringValue(rs.Name)
    isMate := false
    for _, mname := range matenames {
      isMate = isMate || (pkg.FQDN(name) == pkg.FQDN(mname))
    }
    if isMate {
      res = append(res, rs)
    }
  }  
  return res
}