# Osiris - A general purpose, Scale to Zero component for Kubernetes

**This is a fork of the original [deislabs' Osiris](https://github.com/dailymotion/osiris)**.

It was forked before the [HTTPS and HTTP/2 support PR](https://github.com/deislabs/osiris/pull/27),
because we observed [failed requests with the proxy](https://github.com/deislabs/osiris/issues/45)
following this change.

We also have [a set of new features](https://github.com/deislabs/osiris/pulls) we'd like to add,
and we can iterate faster on our own fork.

The long-term plan is NOT to maintain this fork, but to merge back into the original project - 
depending on the original project maintainers.

## Introduction

Osiris enables greater resource efficiency within a Kubernetes cluster by
allowing idling workloads to automatically scale-to-zero and allowing
scaled-to-zero workloads to be automatically re-activated on-demand by inbound
requests.

__Osiris, as a concept, is highly experimental and currently remains under heavy
development.__

## How it works

For Osiris-enabled deployments, Osiris automatically instruments application
pods with a __metrics-collecting proxy__ deployed as a sidecar container.

For any Osiris-enabled deployment that is _already_ scaled to a configurable
minimum number of replicas (one, by default), the __zeroscaler__ component
continuously analyzes metrics from each of that deployment's pods. When the
aggregated metrics reveal that all of the deployment's pods are idling, the
zeroscaler scales the deployment to zero replicas.

Under normal circumstances, scaling a deployment to zero replicas poses a
problem: any services that select pods from that deployment (and only that
deployment) would lose all of their endpoints and become permanently
unavailable. Osiris-enabled services, however, have their endpoints managed by
the Osiris __endpoints controller__ (instead of Kubernetes' built-in endpoints
controller). The Osiris endpoints controller will automatically add Osiris
__activator__ endpoints to any Osiris-enabled service that has lost the rest of
its endpoints.

The Osiris __activator__ component receives traffic for Osiris-enabled services
that are lacking any application endpoints. The activator initiates a scale-up
of a corresponding deployment to a configurable minimum number of replicas (one,
by default). When at least one application pod becomes ready, the request will
be forwarded to the pod.

After the activator "reactivates" the deployment, the __endpoints controller__
(described above) will naturally observe the availability of application
endpoints for any Osiris-enabled services that select those pods and will
remove activator endpoints from that service. All subsequent traffic for the
service will, once again, flow directly to application pods... until a period of
inactivity causes the zeroscaler to take the application offline again.

### Scaling to zero and the HPA

Osiris is designed to work alongside the [Horizontal Pod Autoscaler](https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/) and
is not meant to replace it-- it will scale your pods from n to 0 and from 0 to
n, where n is a configurable minimum number of replicas (one, by default). All
_other_ scaling decisions may be delegated to an HPA, if desired.

This diagram better illustrates the different roles of Osiris, the HPA and the
Cluster Autoscaler:

![diagram](diagram.svg)

## Setup

Prerequisites:

* [Helm](https://helm.sh/docs/intro/) (v2.11.0+, or v3+)
* A running Kubernetes cluster.

### Install Osiris

First, add the Osiris charts repository:

```
helm repo add osiris https://dailymotion.github.io/osiris/charts
```

And then install it:

```
helm install osiris/osiris \
  --name osiris \
  --namespace osiris-system
```

## Usage

Osiris will not affect the normal behavior of any Kubernetes resource without
explicitly being directed to do so.

To enabled the zeroscaler to scale a deployment with idling pods to zero
replicas, annotate the deployment like so:

```
apiVersion: apps/v1
kind: Deployment
metadata:
  namespace: my-aoo
  name: my-app
  annotations:
    osiris.dm.gg/enabled: "true"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-app
  template:
    metadata:
      labels:
        app: nginx
    # ...
  # ...
```

In Kubernetes, there is no direct relationship between deployments and services.
Deployments manage pods and services may select pods managed by one or more
deployments. Rather than attempt to infer relationships between deployments and
services and potentially impact service behavior without explicit consent,
Osiris requires services to explicitly opt-in to management by the Osiris
endpoints controller. Such services must also utilize an annotation to indicate
which deployment should be reactivated when the activator component intercepts a
request on their behalf. For example:

```
kind: Service
apiVersion: v1
metadata:
  namespace: my-namespace
  name: my-app
  annotations:
    osiris.dm.gg/enabled: "true"
    osiris.dm.gg/deployment: my-app
spec:
  selector:
    app: my-app
  # ...
```

### Demo

Deploy the example application `hello-osiris` :

```
kubectl create -f ./example/hello-osiris.yaml
```

This will create an Osiris-enabled deployment and service named `hello-osiris`.

Get the External IP of the `hello-osiris` service once it appears:

```
kubectl get service hello-osiris -o jsonpath='{.status.loadBalancer.ingress[*].ip}'
```

Point your browser to `"http://<EXTERNAL-IP>"`, and verify that
`hello-osiris` is serving traffic.

After about 2.5 minutes, the Osiris-enabled deployment should scale to zero
replicas and the one `hello-osiris` pod should be terminated.

Make a request again, and watch as Osiris scales the deployment back to one
replica and your request is handled successfully.

## Limitations

It is a specific goal of Osiris to enable greater resource efficiency within
Kubernetes clusters, in general, but especially with respect to "nodeless"
Kubernetes options such as
[Virtual Kubelet](https://github.com/virtual-kubelet/virtual-kubelet) or
[Azure Kubernetes Service Virtual Nodes preview](https://docs.microsoft.com/en-us/azure/aks/virtual-nodes-portal), however,
due to known issues with those technologies, Osiris remains incompatible with
them for the near term.

## Contributing

Osiris follows the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md).

## Credit 

[Deislabs](https://github.com/deislabs) for their original work on [Osiris](https://github.com/dailymotion/osiris).
