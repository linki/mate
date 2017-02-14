package producers

import (
	"testing"

	"k8s.io/client-go/pkg/api/v1"
)

func TestValidateService(t *testing.T) {
	loadBalancer := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{v1.LoadBalancerIngress{IP: "8.8.8.8"}},
	}

	emptyService := v1.Service{}

	validService := v1.Service{
		Status: v1.ServiceStatus{LoadBalancer: loadBalancer},
	}

	validMatchedService := v1.Service{
		ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}},
		Status:     v1.ServiceStatus{LoadBalancer: loadBalancer},
	}

	for _, test := range []struct {
		service v1.Service
		filter  map[string]string
		isErr   bool
	}{
		{emptyService, map[string]string{}, false},
		{validService, map[string]string{}, true},
		{validService, map[string]string{"foo": "bar"}, false},
		{validMatchedService, map[string]string{"foo": "bar"}, true},
		{validMatchedService, map[string]string{"foo": "qux"}, false},
	} {
		result := validateService(test.service, test.filter)
		if _, isErr := result.(error); isErr == test.isErr {
			t.Errorf("validateService(%q, %q) => %q, want %t", test.service.Name, test.filter, result, test.isErr)
		}
	}
}
