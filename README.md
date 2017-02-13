# Mate
[![Build Status](https://travis-ci.org/zalando-incubator/mate.svg?branch=master)](https://travis-ci.org/zalando-incubator/mate)
[![Coverage Status](https://coveralls.io/repos/github/zalando-incubator/mate/badge.svg?branch=master)](https://coveralls.io/github/zalando-incubator/mate?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/zalando-incubator/mate)](https://goreportcard.com/report/github.com/zalando-incubator/mate)

# Status

Mate will be merged into new External DNS project - https://github.com/kubernetes-incubator/external-dns, which is meant to be compatible with Mate annotations and become a standard way of creating DNS records for Kubernetes. 

This project is no longer actively developed in the view of External DNS and only minor bug fixes will be considered.  

# Purpose

Mate synchronizes AWS Route53 or Google CloudDNS records with exposed Kubernetes services and ingresses.

When creating ingress objects or services with `Type=LoadBalancer` Kubernetes provisions an external load balancer to forward traffic to its cluster IP. Depending on the cloud provider this load balancer will get a random public IP or DNS name but no defined DNS record to point to it.

Mate bridges this gap and synchronizes DNS records based on the service's or ingress' name and/or namespace. This allows exposed services to be seamlessly discovered even if the load balancer address changes or isn't known beforehand. Additionally, Mate can create DNS entries for each of your `Type=NodePort` services pointing to all nodes in your cluster.

# Install

```
go get github.com/zalando-incubator/mate
```

or

```
docker run registry.opensource.zalan.do/teapot/mate:v0.5.1 --help
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

Depending on the cloud provider the invocation differs slightly. For more detailed step-by-step guide see [examples](examples).

### AWS

```
$ mate \
    --producer=kubernetes \
    --kubernetes-format="{{.Namespace}}-{{.Name}}.example.com" \
    --consumer=aws \
    --aws-record-group-id=foo
```

For each exposed service Mate will create two records in Route53:

1. A record - An Alias to the ELB with the name inferred from `kubernetes-format` or `zalando.org/dnsname` annotation.
 When using ingress DNS records based on the hostnames in your rules will be created.

2. TXT record - A TXT record that will have the same name as an A record and a special identifier with an embedded `aws-record-group-id` value. This helps to identify which records are created via Mate and makes it safe not to overwrite manually created records.

### Google

```
$ mate \
    --producer=kubernetes \
    --kubernetes-format="{{.Namespace}}-{{.Name}}.example.com" \
    --consumer=google \
    --google-project=bar \
    --google-zone=example-com \
    --google-record-group-id=foo
```

Analogous to the AWS case with the difference that it doesn't use the AWS specific Alias functionality but plain A records.

### Permissions

Mate needs permission to modify DNS records in your chosen cloud provider.
On GCP this maps to using service accounts and scopes, on AWS to IAM roles and policies.
For detailed instructions see [the examples](examples).

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
* Creating DNS entries for NodePort services doesn't currently work in combination with the AWS consumer.

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
