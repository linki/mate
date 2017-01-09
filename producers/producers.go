package producers

import (
	"fmt"

	"github.com/zalando-incubator/mate/interfaces"
	"github.com/zalando-incubator/mate/producers/null"
)

func New(name string) (interfaces.Producer, error) {
	switch name {
	case "kubernetes":
		return NewKubernetes()
	case "fake":
		return NewFake()
	case "null":
		return null.NewNull()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
