# mate

Mate creates Google Cloud DNS records for each exposed service.

# Purpose

When creating services with `Type=LoadBalancer` Google will provision an external load balancer to forward traffic to your service. This load balancer will get a public IP but no DNS record to point to it.

Mate bridges this gap and creates and syncs Google Cloud DNS records based on the service's name and/or namespace. This allows services to seamlessly being discovered even if their IPs change.

# Caveats

Mate currently only supports Service objects that use `Type=LoadBalancer`, i.e. `Ingress` is not supported.
