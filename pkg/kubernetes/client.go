package kubernetes

import (
	"net/url"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// NewClient configures a new Kubernetes client. If apiServerURL is nil the
// client with be configured for in-cluster-use.
func NewClient(apiServerURL *url.URL) (*kubernetes.Clientset, error) {
	var config *rest.Config
	if apiServerURL == nil {
		var err error
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		config = &rest.Config{
			Host: apiServerURL.String(),
		}
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return client, nil
}
