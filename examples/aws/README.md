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
*Note*: `example.com` from the manfiest should be changed to the real hosted zone existing in your AWS account. 
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
Running `kubectl create -f service.yaml` will create a service in default namepsace and AWS will provision an ELB pointing to the service.
Shortly Mate will create a DNS record as according to the specified template (namespace-name.hosted-zone) pointing to the provisioned ELB: 
`default-nginx-service.example.com`. 

If you have `aws-cli` installed, this can be verified with (change `example.com` to your real hosted zone):

`aws route53 list-resource-record-sets --hosted-zone-id=*my-zone-id* --query "ResourceRecordSets[?Name == 'default-nginx-service.example.com.']"` 

If you do not wish to use a template approach, or want to create a record in another hosted-zone, this can be achieved by specifying desired DNS 
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
Once LB is created Mate will create a DNS records, as specified in `rules.Host` fields, e.g. in the specified example it will create 
two records in two separate hosted zones `bar-app.example.com` and `foo-app.foo.com`. 

