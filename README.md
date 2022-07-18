# jiffykube
K8s API built using KubeBuilder that I hack on to experiment with networking and custom resources

Creating an application will create a deployment, service, and ingress.

## Getting started w/ development version
- Run a local kuberentes cluster, e.g. minikube: https://minikube.sigs.k8s.io/docs/start/
  - assure that you have loadbalancing enabled, e.g. `minikube tunnel`
- `make generate manifests install`
- Run jiffykube: `go run main.go --enable-leader-election=false --disable-webhooks
- Deploy an application: `kubectl apply -f myapp.yaml`

myapp.yaml:
```
apiVersion: base.jiffykube.io/v1
kind: App
metadata:
  name: apple
spec:
  name: apple
  replicas: 2
  containers:
    - name: sample
      image: gcr.io/kubernetes-312803/sample-go-app
      ports:
        containerPort: 8888
  rules:
    - path: "/"
  ingressClass: "default"
```
