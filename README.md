# Mate
[![Build Status](https://travis-ci.org/zalando-incubator/mate.svg?branch=master)](https://travis-ci.org/zalando-incubator/mate)

Mate synchronizes AWS Route53 or Google CloudDNS records with exposed Kubernetes services and ingresses.

# Purpose

When creating ingress objects or services with `Type=LoadBalancer` Kubernetes provisions an external load balancer to forward traffic to its cluster IP. Depending on the cloud provider this load balancer will get a random public IP or DNS name but no defined DNS record to point to it.

Mate bridges this gap and synchronizes DNS records based on the service's or ingress' name and/or namespace. This allows exposed services to be seamlessly discovered even if the load balancer address changes or isn't known beforehand.

# Install

```
go get github.com/zalando-incubator/mate
```

or

```
docker run registry.opensource.zalan.do/teapot/mate:v0.3.0 --help
```

# Features

1. Supports both Google and Amazon Cloud Providers
2. Complete and safe management of DNS records for both services and ingress resources. Only records created by Mate
will be updated and deleted. 
3. Immediate updates via Kubernetes event listener and periodic resync of all endpoints to match the Kubernetes state.
4. Allows to specify record DNS via Service Annotations, Ingress Rules or passed in go-template 
5. Pluggable consumers and producers (see below)
6. Supports multiple hosted zones in AWS Route53

# Usage

Depending on the cloud provider the invocation differs slightly. For more detailed step-by-step guide see [examples](mate/tree/master/examples). 

### AWS

```
$ mate \
    --producer=kubernetes \
    --kubernetes-format="{{.Namespace}}-{{.Name}}.example.com" \
    --consumer=aws \
    --aws-record-group-id=foo
```

For each exposed service Mate will create two records in Route53:

1. A record - An Alias to the ELB with the name inferred from `kubernetes-format` and `kubernetes-domain`. So if you create an nginx service named `my-nginx` in the `default` namespace and use a `example.com` as domain the registered record will have a hostname of `default-my-nginx.example.com`. You can, however, overwrite the generated DNS name by using an annotation on the service (`zalando.org/dnsname`). When using ingress DNS records based on the hostnames in your rules will be created.

2. TXT record - A TXT record that will have the same name as an A record (`default-my-nginx.example.com`) and a special identifier with an embedded `aws-record-group-id` value. This helps to identify which records are created via Mate and makes it safe not to overwrite manually created records.

### Google

```
$ mate \
    --producer=kubernetes \
    --kubernetes-format="{{.Namespace}}-{{.Name}}.example.com" \
    --consumer=google \
    --google-project=bar \
    --google-zone=example-com
    --google-record-group-id=foo
```

Analogous to the AWS case with the difference that it doesn't use the AWS specific Alias functionality but plain A records.

### Permissions

`Mate` needs permission to modify DNS records in your chosen cloud provider.
On GCP this maps to using service accounts and scopes, on AWS to IAM roles and policies.

#### AWS

By default, `mate` runs under the same IAM role as the cluster node that the instance
of `mate` currently runs on. If you want to restrict the permissions `mate` gets, you can use [`kube2iam`](https://github.com/jtblin/kube2iam) to set a different IAM role as context for each of your Pods in general.

Follow the instructions in the [`kube2iam`](https://github.com/jtblin/kube2iam#kube2iam) docs on how to deploy it in your cluster.
Then create a new IAM role specifically for `mate` with enough permissions to manage your DNS records. We use the following:

```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Action": "route53:*",
            "Resource": "*",
            "Effect": "Allow"
        },
        {
            "Action": "elasticloadbalancing:DescribeLoadBalancers",
            "Resource": "*",
            "Effect": "Allow"
        }
    ]
}
```

You also need your new role to be assumed by whatever IAM role your worker nodes run under,
see [Trust Relationship](https://github.com/jtblin/kube2iam#iam-roles) in the `kube2iam` docs for that. Your Trust Relationship for the `mate` IAM role should look similar to this:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    },
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "arn:aws:iam::<AWS_ACCOUNT_ID>:role/kube-worker-node"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

With that being setup you can annotate your Pods with the name of the role you want to have it run under, e.g., assuming you named your IAM role `mate`, like this:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mate
spec:
  template:
    metadata:
      ...
      annotations:
        iam.amazonaws.com/role: mate
    spec:
      containers:
      - name: mate
      ...
```

#### Google

By default, `mate` runs with the same scopes as the cluster node that the instance
of `mate` currently runs on.
`Mate` needs the `https://www.googleapis.com/auth/ndev.clouddns.readwrite` scope in order
to manage your DNS records. When creating a new GKE cluster or node pool you can pass this
value as an argument to `--scopes`.

```
$ gcloud container node-pools create "my-pool" \
    --cluster "my-cluster" \
    --scopes "https://www.googleapis.com/auth/ndev.clouddns.readwrite"
```

Make sure the nodes that get created have the correct scopes set and that `mate` runs on them.

```
gcloud compute instances describe gke-your-node --format json | jq '.serviceAccounts[].scopes'
```

You can also force `mate` to use a specific service account by providing it a service account
credentials file via a mounted secret. First create a service account and give it a role
permissive enough to manage your DNS records. (TODO: find out which one is suited best.)
Then create a key and download the corresponding json credentials file, e.g. with:

```
$ gcloud iam service-accounts keys create gcloud-credentials.json --iam-account service-account-name@project-id.iam.gserviceaccount.com
```

Create a Kubernetes secret based on that file, e.g. with:

```
$ kubectl create secret generic gcloud-config --from-file=gcloud-credentials.json
```

You can then mount that secret into the `mate` container and pass it the `GOOGLE_APPLICATION_CREDENTIALS` environment variable pointing to the location of the file so that the Go client will find it with:

```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mate
spec:
  template:
    ...
    spec:
      containers:
      - name: mate
        ...
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: /etc/mate/gcloud-credentials.json
        volumeMounts:
        - mountPath: /etc/mate
          name: gcloud-config
          readOnly: true
      volumes:
      - name: gcloud-config
        secret:
          secretName: gcloud-config
```

### Kubernetes

Mate will listen for events from the API Server and create corresponding
records for newly created services. Further synchronization (create, update and
removal) will occur every minute. There's an initial syncronization when Mate
boots up so it's safe to reboot the process at any point in time. If you only
like to do the synchronization you can use the `sync-only` flag.

By default Mate uses the in cluster environment to configure the connection to
the API server. When running outside a cluster it is possible to configure the
API server url using the flag `kubernetes-server`. For instance you can run
Mate locally with the server URL set to `http://127.0.0.1:8001` and use
`kubectl proxy` to forward requests to a cluster.

# Producers and Consumers

Mate supports swapping out Endpoint producers (e.g. a service list from Kubernetes) and endpoint consumers (e.g. making API calls to Google to create DNS records) and both sides are pluggable. There currently exist two producer and three consumer implementations.

### Producers

* `Kubernetes`: watches kubernetes services and ingresses with exactly one external IP or DNS name
* `Fake`: generates random endpoints simulating a very busy cluster

### Consumers

* `Google`: listens for endpoints and creates Google CloudDNS entries accordingly
* `AWS`   : listens for endpoints and creates AWS Route53 DNS entries
* `Stdout`: listens for endpoints and prints them to Stdout

You can choose any combination. `Kubernetes` + `Stdout` is useful for testing your service watching functionality, whereas `Fake` + `Google` is useful for testing that you create the correct records in GCP.

# Caveats

* Although the modular design allows to specify it, you currently cannot create DNS records on Google CloudDNS for a cluster running on AWS because AWS ELBs will send ELB endpoints in the form of DNS names whereas the Google consumer expects them to be IPs and vice versa.

# License

The MIT License (MIT)

Copyright (c) 2016 Zalando SE

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
