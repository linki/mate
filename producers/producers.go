package producers

import (
	"fmt"

	"github.com/zalando-incubator/mate/interfaces"
	"github.com/zalando-incubator/mate/producers/fake"
	"github.com/zalando-incubator/mate/producers/kubernetes"
)

func New(name string) (interfaces.Producer, error) {
	switch name {
	case "kubernetes":
		return kubernetes.NewProducer()
	case "fake":
		return fake.NewFake()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
