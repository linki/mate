package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

var (
	evaluateTargetHealth = true
	defaultTxtTTL        = int64(300)
)

func (c *Client) initRoute53Client() (*route53.Route53, error) {
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

	return route53.New(session), nil
}

func createChangesList(action string, rsets []*route53.ResourceRecordSet) []*route53.Change {
	var changes []*route53.Change
	for _, rset := range rsets {
		changes = append(changes, &route53.Change{
			Action:            aws.String(action),
			ResourceRecordSet: rset,
		})
	}
	return changes
}
