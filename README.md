# service-operator
service-operator is Kubernetes operator with a Custom Resource for easily deploying web services.

## Description
This repo exposes the `apps.ghaabor.io/v1` Kubernetes API and currently it contains one CR called `WebService`. The Go code implements the Controller and the respective Go types for `WebService` resource.

The `WebService` resource exposes a simple API:

```yaml
apiVersion: apps.ghaabor.io/v1
kind: WebService
metadata:
  name: webservice-sample
spec:
  replicas: 1
  host: "demo.ghaabor.io"
  image: "nginx:latest"
```

In the background, the following Kubernetes resources are created after apply:

* `Deployment`: Using the provided `replicas` and `image`. Image can be any docker image which exposes something on the port `80`.
* `Service`: Exposes the deployment's port `80`.
* `Ingress`: A Kubernetes `Ingress` using the NGINX Ingress Controller with the hostname defined in `host`, with TLS enabled for it using `cert-manager` (see later in _Cluster prerequisites_) and it's attached to the previously created `Service` resource.

## Getting Started
You’ll need a Kubernetes cluster to run against. You can use something like [KIND](https://sigs.k8s.io/kind), [minikube](https://minikube.sigs.k8s.io/docs/) or [Docker Desktop](https://docs.docker.com/desktop/kubernetes/) to get a local cluster for testing, or run against a remote cluster.

**NOTE:** Your controller will automatically use the current context in your kubeconfig file (i.e. whatever cluster `kubectl cluster-info` shows).

### Cluster prerequisites
* [NGINX Ingress Controller](https://kubernetes.github.io/ingress-nginx/deploy/#quick-start)
* [cert-manager installed](https://cert-manager.io/docs/installation/kubectl/)
* [letsencrypt issuer configured](https://cert-manager.io/docs/tutorials/acme/nginx-ingress/#step-6---configure-a-lets-encrypt-issuer)
* Modify your `/etc/hosts` file and add: `127.0.0.1 demo.ghaabor.io`
    * Another option is to run e.g. `curl` commands with an addition host header: `curl -H 'Host: demo.ghaabor.io' http://localhost`

**NOTE:** Deploying the operator on a remote setup requires a proper DNS configuration which is not covered in this documentation.

### Example letsencrypt issuer

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt
spec:
  selfSigned: {}
```

Or [create your own CA](https://medium.com/nerd-for-tech/adventures-in-encryption-securing-your-laptop-kubernetes-cluster-9e032bf77f3e) and use that to generate keys:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: letsencrypt
spec:
  ca:
    secretName: supersecret-ca-keypair
```

**NOTE:** When using your own CA, don't forget to import the certificate into your browser of choice!

### Running on the cluster
1. Deploy the controller to the cluster with the image specified by `IMG`:

```sh
make deploy IMG=ghcr.io/ghaabor/service-operator:main
```

2. Install Instances of Custom Resources:

```sh
kubectl apply -f config/samples/
```

3. Open [https://demo.ghaabor.io](https://demo.ghaabor.io)

### Undeploy controller
Remove the controller from the cluster:

```sh
make undeploy
```

## How it works
This project aims to follow the Kubernetes [Operator pattern](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/)

It uses [Controllers](https://kubernetes.io/docs/concepts/architecture/controller/) 
which provides a reconcile function responsible for synchronizing resources untile the desired state is reached on the cluster 

### Test It Out
1. Install the CRDs into the cluster:

```sh
make install
```

2. Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```sh
make run
```

**NOTE:** You can also run this in one step by running: `make install run`

### Uninstall CRDs
To delete the CRDs from the cluster:

```sh
make uninstall
```

### Modifying the API definitions
If you are editing the API definitions, generate the manifests such as CRs or CRDs using:

```sh
make manifests
```

**NOTE:** Run `make --help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

