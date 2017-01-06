package null

import (
	"testing"

	"github.com/zalando-incubator/mate/pkg"
)

func TestNullReturnsZeroEndpoints(t *testing.T) {
	endpoints := newEndpoints(t, nil)

	count := len(endpoints)
	if count != 0 {
		t.Error(count)
	}
}

func newEndpoints(t *testing.T, producer *nullProducer) []*pkg.Endpoint {
	if producer == nil {
		producer = &nullProducer{}
	}

	endpoints, err := producer.Endpoints()
	if err != nil {
		t.Fatal(err.Error())
	}

	return endpoints
}
