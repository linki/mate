package test

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
)

func GetHostedZones() map[string]string {
	return map[string]string{
		"example.com.":     "example.com.",
		"foo.com.":         "foo.com.",
		"sub.example.com.": "sub.example.com.",
	}
}

func GetOriginalState(groupID string) map[string][]*route53.ResourceRecordSet {
	return map[string][]*route53.ResourceRecordSet{
		"foo.com.": []*route53.ResourceRecordSet{
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("public-ip.foo.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String("127.0.0.1"),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("public-ip.foo.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String(groupID),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("test.foo.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("404.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("update.foo.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("404.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("update.foo.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String(groupID),
					},
				},
			},
		},
		"example.com.": []*route53.ResourceRecordSet{
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("public-ip.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String("192.168.0.1"),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("test.example.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("404.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("test.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String(groupID),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("update.example.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("302.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("update.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String(groupID),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("another.example.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("200.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("another.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String("ignored"),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("CNAME"),
				Name: aws.String("cname.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String("some-elb.amazon.com"),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("TXT"),
				Name: aws.String("withoutA.example.com."),
				ResourceRecords: []*route53.ResourceRecord{
					&route53.ResourceRecord{
						Value: aws.String("lonely"),
					},
				},
			},
			&route53.ResourceRecordSet{
				Type: aws.String("A"),
				Name: aws.String("withouttxt.example.com."),
				AliasTarget: &route53.AliasTarget{
					DNSName:      aws.String("random.elb.com"),
					HostedZoneId: aws.String("123"),
				},
			},
		},
	}
}
