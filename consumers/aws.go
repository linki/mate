package consumers

import "github.bus.zalan.do/teapot/mate/pkg"

type awsClient interface {
	ListRecordSets() ([]*pkg.Endpoint, error)
	ChangeRecordSets(upsert []*pkg.Endpoint, delete []*pkg.Endpoint) error
}

type aws struct{}

func (a *aws) Sync([]*pkg.Endpoint) error  { return nil }
func (a *aws) Process(*pkg.Endpoint) error { return nil }
