# mate

Mate creates Google Cloud DNS records for each exposed service.

# Purpose

When creating services with `Type=LoadBalancer` Google will provision an external load balancer to forward traffic to your service. This load balancer will get a public IP but no DNS record to point to it.

Mate bridges this gap and creates and syncs Google Cloud DNS records based on the service's name and/or namespace. This allows services to seamlessly being discovered even if their IPs change.

# Usage

```
$ mate \
    --producer=kubernetes \
    --kubernetes-domain=oolong.gcp.zalan.do \
    --kubernetes-format="{{.Namespace}}-{{.Name}}" \
    --consumer=google \
    --google-project=zalando-teapot \
    --google-zone=oolong-gcp-zalan-do
```

This will watch the kubernetes endpoint at `http://127.0.0.1:8080` for services and manage DNS records that are defined in a zone called `oolong-gcp-zalan-do` in project `zalando-teapot`.
It will be "authoritive" for any A records that contain `oolong.gcp.zalan.do`, which means it will add, update **and delete** records to keep them in sync with exposed services.
The format of the created DNS records can be specific via Go templates as seen above.
So, in this case, a service called `apiserver` in namespace `prod` would result in an A record `prod-apiserver.oolong.gcp.zalan.do` pointing to its external IP.

# Producers and Consumers

Mate supports swapping out Endpoint producers (e.g. a service list from Kubernetes) and endpoint consumers (e.g. making API calls to Google to create DNS records) and both sides are pluggable. There currently exist four implementations, two for each side.

### Producers
* `Kubernetes`: watches kubernetes services with exactly one external IP
* `Fake`: generates random endpoints simulating a very busy cluster

### Consumers
* `Google`: listens for endpoints and creates Google Cloud DNS entries accordingly
* `AWS`   : listens for endpoints and creates AWS Route53 DNS entries 
* `Stdout`: listens for endpoints and prints them to Stdout

You can choose any combination. `Kubernetes` + `Stdout` is useful for testing your service watching functionality, whereas `Fake` + `Google` is useful for testing that you create the correct records in GCP.

# Caveats

Mate currently only supports Service objects that use `Type=LoadBalancer`, i.e. `Ingress` is not supported.
