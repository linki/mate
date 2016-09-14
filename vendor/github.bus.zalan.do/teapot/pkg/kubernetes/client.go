package kubernetes

import (
	"fmt"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"

	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
)

// NewClient returns a Kubernetes API client for a given endpoint.
func NewClient(server *url.URL) (*unversioned.Client, error) {
	client, err := unversioned.New(&restclient.Config{
		Host: server.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("Unable to create Kubernetes API client: %v", err)
	}
	return client, nil
}

// NewClientOrDie returns a Kubernetes API client for a given endpoint.
// In case of error it exits the process.
func NewClientOrDie(server *url.URL) *unversioned.Client {
	client, err := NewClient(server)
	if err != nil {
		log.Fatal(err)
	}
	return client
}

func NewHealthyClient(server *url.URL) *unversioned.Client {
	return Wait(NewClientOrDie(server))
}

func IsHealthy(client *unversioned.Client) bool {
	_, err := client.ServerVersion()
	return err == nil
}

func Wait(client *unversioned.Client) *unversioned.Client {
	for {
		if IsHealthy(client) {
			return client
		}

		log.Infoln("Kubernetes API server is unavailable. Waiting...")
		time.Sleep(10 * time.Second)
	}
}
