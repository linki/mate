package producers

import (
	"testing"

	"k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestValidateIngress(t *testing.T) {
	loadBalancer := v1.LoadBalancerStatus{
		Ingress: []v1.LoadBalancerIngress{v1.LoadBalancerIngress{IP: "8.8.8.8"}},
	}

	emptyIngress := extensions.Ingress{}

	validIngress := extensions.Ingress{
		Status: extensions.IngressStatus{LoadBalancer: loadBalancer},
	}

	validMatchedIngress := extensions.Ingress{
		ObjectMeta: v1.ObjectMeta{Annotations: map[string]string{"foo": "bar"}},
		Status:     extensions.IngressStatus{LoadBalancer: loadBalancer},
	}

	for _, test := range []struct {
		ingress extensions.Ingress
		filter  map[string]string
		isErr   bool
	}{
		{emptyIngress, map[string]string{}, false},
		{validIngress, map[string]string{}, true},
		{validIngress, map[string]string{"foo": "bar"}, false},
		{validMatchedIngress, map[string]string{"foo": "bar"}, true},
		{validMatchedIngress, map[string]string{"foo": "qux"}, false},
	} {
		result := validateIngress(test.ingress, test.filter)
		if _, isErr := result.(error); isErr == test.isErr {
			t.Errorf("validateIngress(%q, %q) => %q, want %t", test.ingress.Name, test.filter, result, test.isErr)
		}
	}
}
