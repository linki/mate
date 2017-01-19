package deploy

import (
	"fmt"
	"net/url"

	"k8s.io/client-go/pkg/api/resource"
	"k8s.io/client-go/pkg/api/unversioned"
	api "k8s.io/client-go/pkg/api/v1"
	extensions "k8s.io/client-go/pkg/apis/extensions/v1beta1"

	"github.com/zalando-incubator/mate/pkg/kubernetes"
)

type manifest struct {
	version string
	args    []string
}

func NewManifest(version string, args []string) *manifest {
	for i, val := range args {
		if val == "--deploy" {
			args = append(args[:i], args[i+1:]...)
		}
	}

	return &manifest{
		version: version,
		args:    args,
	}
}

func (m *manifest) Deploy() error {
	var manifest = &extensions.Deployment{
		TypeMeta: unversioned.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Deployment",
		},
		ObjectMeta: api.ObjectMeta{
			Name:      "mate",
			Namespace: "default",
			Labels: map[string]string{
				"application": "mate",
				"version":     m.version,
			},
		},
		Spec: extensions.DeploymentSpec{
			Selector: &unversioned.LabelSelector{
				MatchLabels: map[string]string{
					"application": "mate",
				},
			},
			Template: api.PodTemplateSpec{
				ObjectMeta: api.ObjectMeta{
					Labels: map[string]string{
						"application": "mate",
						"version":     m.version,
					},
					Annotations: map[string]string{
						"scheduler.alpha.kubernetes.io/critical-pod": "",
						"scheduler.alpha.kubernetes.io/tolerations":  `[{"key":"CriticalAddonsOnly", "operator":"Exists"}]`,
						"iam.amazonaws.com/role":                     "kube-aws-test-linki39-app-mate",
					},
				},
				Spec: api.PodSpec{
					Containers: []api.Container{
						{
							Name:  "mate",
							Image: "registry.opensource.zalan.do/teapot/mate:" + m.version,
							Args:  m.args,
							Env: []api.EnvVar{
								{
									Name:  "AWS_REGION",
									Value: "eu-central-1",
								},
							},
							Resources: api.ResourceRequirements{
								Requests: api.ResourceList{
									"cpu":    resource.MustParse("50m"),
									"memory": resource.MustParse("25Mi"),
								},
								Limits: api.ResourceList{
									"cpu":    resource.MustParse("200m"),
									"memory": resource.MustParse("200Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	url, _ := url.Parse("http://127.0.0.1:8001")

	client, err := kubernetes.NewClient(url)
	if err != nil {
		return fmt.Errorf("Unable to setup Kubernetes API client: %v", err)
	}

	_, err = client.Extensions().Deployments(api.NamespaceDefault).Update(manifest)
	if err != nil {
		return fmt.Errorf("Unable to submit manifest: %v", err)
	}

	return nil
}
