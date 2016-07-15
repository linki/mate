# mate

Mate creates Google Cloud DNS records for each exposed service.

# Purpose

When creating services with `Type=LoadBalancer` Google will provision an external load balancer to forward traffic to your service. This load balancer will get a public IP but no DNS record to point to it.

Mate bridges this gap and creates and syncs Google Cloud DNS records based on the service's name and/or namespace. This allows services to seamlessly being discovered even if their IPs change.

# Usage

```
$ mate \
    --master=http://127.0.0.1:8080 \
    --project=zalando-teapot \
    --zone=gcp-zalan-do \
    --domain=oolong.gcp.zalan.do \
    --format={{.Namespace}}-{{.Name}}.{{.Domain}}.
```

This will watch the kubernetes endpoint at `http://127.0.0.1:8080` for services and manage DNS records that are defined in a zone called `gcp-zalan-do` in project `zalando-teapot`.
It will be "authoritive" for any A records that contain `oolong.gcp.zalan.do`, which means it will add, update **and delete** records to keep them in sync with exposed services.
The format of the created DNS records can be specific via Go templates as seen above.
So, in this case, a service called `apiserver` in namespace `prod` would result in an A record `prod-apiserver.oolong.gcp.zalan.do` pointing to its external IP.

# Caveats

Mate currently only supports Service objects that use `Type=LoadBalancer`, i.e. `Ingress` is not supported.
