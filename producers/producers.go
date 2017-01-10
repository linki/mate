package producers

import (
	"fmt"

	"github.com/zalando-incubator/mate/interfaces"
)

func New(name string) (interfaces.Producer, error) {
	switch name {
	case "kubernetes":
		return NewKubernetes()
	case "fake":
		return NewFake()
	case "null":
		return NewNull()
	}
	return nil, fmt.Errorf("Unknown producer '%s'.", name)
}
