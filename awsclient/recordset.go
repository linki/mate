package awsclient

import (
	"github.bus.zalan.do/teapot/mate/pkg"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/aws/aws-sdk-go/aws"
	"fmt"
)

//handle upsert and delete cases separately for ease of understanding
func (c *Client) upsertRecords(zoneID *string, eps []*pkg.Endpoint) []*route53.Change {
  var changes []*route53.Change
  for _, ep := range eps {
    changes = append(changes, &route53.Change{
      Action: aws.String("UPSERT"),
      ResourceRecordSet: ep.AWSARecordAlias(zoneID, int64(c.options.RecordSetTTL)),
    })
    changes = append(changes, &route53.Change{
      Action: aws.String("UPSERT"),
      ResourceRecordSet: ep.AWSTXTRecord(int64(c.options.RecordSetTTL)),
    })
  }
  return changes
}

func (c *Client) deleteRecords(zoneID *string, clusterName string, eps []*pkg.Endpoint) ([]*route53.Change, error) {
  allrs, err := c.getRecordSets()
  if err != nil {
    return nil, err
  }
  //filter out TXT records to extract hostnames belonging to mate
  matenames := make([]string, 0, 0)
  changes := make([]*route53.Change, 0, 0)
  for _, rs := range allrs {
    if aws.StringValue(rs.Type) == "TXT" && len(rs.ResourceRecords) == 1 {
      resource := rs.ResourceRecords[0]
      if aws.StringValue(resource.Value) == pkg.GetMateValue(clusterName) {
        matenames = append(matenames, *rs.Name)
      }
    }
  }
  for _, ep := range eps {
    dns := ep.DNSName
    isMate := false
    for _, name := range matenames {
      isMate = isMate || (pkg.FQDN(dns) == pkg.FQDN(name))
    }
    if isMate {
      changes = append(changes, &route53.Change{
        Action: aws.String("UPSERT"),
        ResourceRecordSet: ep.AWSARecordAlias(zoneID, int64(c.options.RecordSetTTL)),
      })
      changes = append(changes, &route53.Change{
        Action: aws.String("UPSERT"),
        ResourceRecordSet: ep.AWSTXTRecord(int64(c.options.RecordSetTTL)),
      })      
    }
  }
  return changes, nil
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
