# Mate

Mate synchronizes AWS Route53 or Google CloudDNS records with exposed Kubernetes services.

# Purpose

When creating services with `Type=LoadBalancer` Kubernetes provisions an external load balancer to forward traffic to your service. Depending on the cloud provider this load balancer will get a random public IP or DNS name but no defined DNS record to point to it.

Mate bridges this gap and synchronizes DNS records based on the service's name and/or namespace. This allows services to be seamlessly discovered even if the load balancer address changes.

# Install

```
go get github.com/zalando-incubator/mate
```

or

```
docker run registry.opensource.zalan.do/teapot/mate:v0.0.7 --help
```

# Usage

Depending on the cloud provider the invocation differs slightly

### AWS

```
$ mate \
    --producer=kubernetes \
    --kubernetes-domain=example.com \
    --kubernetes-format="{{.Namespace}}-{{.Name}}" \
    --consumer=aws \
    --aws-hosted-zone=example.com. \
    --aws-record-group-id=foo
```

For each exposed service Mate will create two records in Route53: 

1. A record - Alias to the ELB with the name inferred from `kubernetes-format` and `kubernetes-domain`. So if you create an nginx service named `my-nginx` in `default` namespace and domain `example.com` registered record will have a hostname `default-my-nginx.example.com`. 
 
2. TXT record - which will have the same name as an A record (`default-my-nginx.example.com`) and a special identifier with embedded `aws-record-group-id` value. TXT record helps to identify which records are created via Mate and makes it safe not to overwrite manually created records 

### Google

```
$ mate \
    --producer=kubernetes \
    --kubernetes-domain=example.com \
    --kubernetes-format="{{.Namespace}}-{{.Name}}" \
    --consumer=google \
    --google-project=bar \
    --google-zone=example-com
    --google-record-group-id=foo
```

Analogous to the AWS case

### Kubernetes

By default Mate will retrieve the list of services from Kubernetes API server via `http://127.0.0.1:8001` (for local testing use `kubectl proxy`), however API server url can be configured with `kubernetes-server` flag. 
Mate will listen for events from Kubernetes API Server and create corresponding records for newly created services. Further synchronization (create, removal, update) will occur every minute. If you only like to do the synchronization you can enable `sync-only` flag. 

# Producers and Consumers

Mate supports swapping out Endpoint producers (e.g. a service list from Kubernetes) and endpoint consumers (e.g. making API calls to Google to create DNS records) and both sides are pluggable. There currently exist two producer and three consumer implementations.

### Producers

* `Kubernetes`: watches kubernetes services with exactly one external IP
* `Fake`: generates random endpoints simulating a very busy cluster

### Consumers

* `Google`: listens for endpoints and creates Google CloudDNS entries accordingly
* `AWS`   : listens for endpoints and creates AWS Route53 DNS entries 
* `Stdout`: listens for endpoints and prints them to Stdout

You can choose any combination. `Kubernetes` + `Stdout` is useful for testing your service watching functionality, whereas `Fake` + `Google` is useful for testing that you create the correct records in GCP.

# Caveats

* Mate currently only supports Service objects that use `Type=LoadBalancer`, i.e. `Ingress` is not supported.
* Although the modular design allows to specify it, you currently cannot create DNS records on Google CloudDNS for a cluster running on AWS because AWS ELBs will send ELB endpoints in the form of DNS names whereas the Google consumer expects them to be IPs and vice versa.
