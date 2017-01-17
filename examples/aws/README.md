# Running Mate on AWS
The following examples show how to use Mate to create DNS entries in Route53 for ingress and services.
In all of the examples mate is deployed as a kubernetes deployment:

```
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: mate
spec:
  replicas: 1
  template:
    metadata:
      labels:
        app: mate
    spec:
      containers:
      - name: mate
        image: registry.opensource.zalan.do/teapot/mate:v0.4.0
        args:
        - --producer=kubernetes
        - --kubernetes-format={{.Namespace}}-{{.Name}}.example.com
        - --consumer=aws
        - --aws-record-group-id=my-cluster
```
*Note*: `example.com` is a hosted zone where records are created. Mate does not create hosted zones, and it assumes there exist at least one hosted zone where the record with a given DNS can be placed.
## Service

Create a service using the following manifest [service](service.yaml):
```
apiVersion: v1
kind: Service
metadata:
  name: nginx-service
  labels:
    app: nginx
spec:
  type: LoadBalancer
  ports:
  - port: 80
    targetPort: 80
  selector:
    app: behind-nginx-app
```
Running `kubectl create -f service.yaml` will create a service in default namespace and AWS will provision an ELB pointing to the service.
Shortly Mate will create a DNS record as according to the specified template (namespace-name.hosted-zone) pointing to the provisioned ELB:
`default-nginx-service.example.com`.

If you have `aws-cli` installed, this can be verified with (change `example.com` to your real hosted zone):

`aws route53 list-resource-record-sets --hosted-zone-id=*my-zone-id* --query "ResourceRecordSets[?Name == 'default-nginx-service.example.com.']"`

If you do not wish to use a template approach, or want to create a record in another hosted-zone (different from specified in Mate deployment manifest), this can be achieved by specifying desired DNS
in service annotations, e.g.:
```
...
metadata:
  name: nginx-service
  annotations:
    zalando.org/dnsname: annotated-nginx.foo.com
...
```

## Ingress
Use the following example to create an ingress:

```
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: test
spec:
  rules:
  - host: foo-app.foo.com
    http:
      paths:
      - path: /foo
        backend:
          serviceName: fooSvc
          servicePort: 80
  - host: bar-app.example.com
    http:
      paths:
      - path: /bar
        backend:
          serviceName: barSvc
          servicePort: 80
```

*Note*: To use kubernetes ingress on AWS you need to run an Ingress controller in your cluster. Possible implementation details can be found here:
https://github.com/kubernetes/contrib/tree/master/ingress/controllers


Your Ingress controller should provision a Load Balancer (both ELB and ALB are supported by Mate) and update the ingress resource.
Once LB is created Mate will create a DNS records, as specified in `rules.Host` field of created ingress resource, e.g. in the specified example it will create
two records in two separate hosted zones `bar-app.example.com` and `foo-app.foo.com` (assuming both exist in your AWS account).

### Permissions

By default, `mate` runs under the same IAM role as the cluster node that the instance of `mate` currently runs on.
If you want to restrict the permissions `mate` gets, you can use [`kube2iam`](https://github.com/jtblin/kube2iam)
to set a different IAM role as context for each of your Pods in general.

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
