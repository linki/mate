# Running Mate on Google
The following examples show how to use Mate to create DNS entries in Google CloudDNS for ingress and services.
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
        image: registry.opensource.zalan.do/teapot/mate:v0.3.0
        args:
        - --producer=kubernetes
        - --kubernetes-format={{.Namespace}}-{{.Name}}.example.com
        - --consumer=google
        - --google-project=my-project
        - --google-zone=example-com
        - --google-record-group-id=my-cluster
```
*Note*: `example.com` from the manfiest should be changed to the real hosted zone existing in your Google account.
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
Running `kubectl create -f service.yaml` will create a service in default namepsace and Google will provision a load balancer pointing to the service.
Shortly Mate will create a DNS record as according to the specified template (namespace-name.hosted-zone) pointing to the provisioned load balancer:
`default-nginx-service.example.com`.

If you have `gcloud` installed, this can be verified with (change `example.com` to your real hosted zone):

`gcloud dns record-sets list --zone example-com --filter default-nginx-service.example.com.`

If you do not wish to use a template approach, this can be achieved by specifying desired DNS
in service annotations, e.g.:
```
...
metadata:
  name: nginx-service
  annotations:
    zalando.org/dnsname: annotated-nginx.example.com
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
  - host: foo-app.example.com
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

*Note*: To use kubernetes ingress on Google you need to run an Ingress controller in your cluster. Possible implementation details can be found here:
https://github.com/kubernetes/contrib/tree/master/ingress/controllers. On GKE this is usually run for you by default.


Your Ingress controller should provision a Load Balancer and update the ingress resource.
Once LB is created Mate will create a DNS records, as specified in `rules.Host` fields, e.g. in the specified example it will create
two records in the hosted zone: `bar-app.example.com` and `foo-app.example.com`.

### Permissions

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
