package kubernetes

import (
	"log"
	"time"

	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/client/unversioned"
)

func NewClient(server string) (*unversioned.Client, error) {
	client, err := unversioned.New(&restclient.Config{
		Host: server,
	})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func NewClientOrDie(server string) *unversioned.Client {
	client, err := NewClient(server)
	if err != nil {
		log.Fatal(err)
	}

	return client
}

func NewHealthyClient(server string) *unversioned.Client {
	return Wait(NewClientOrDie(server))
}

func IsHealthy(c *unversioned.Client) bool {
	if _, err := c.ServerVersion(); err != nil {
		return false
	}

	return true
}

func Wait(c *unversioned.Client) *unversioned.Client {
	for {
		if IsHealthy(c) {
			return c
		}

		log.Printf("Kubernetes API server is unavailable. Waiting...")
		time.Sleep(10 * time.Second)
	}
}
