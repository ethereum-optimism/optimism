# kovan-replica-deploy

## Prerequisites

- `kubectl`
- `kustomize` from a recent [kustomize release](https://github.com/kubernetes-sigs/kustomize/releases/)
- `kind` from a recent [kind release](https://github.com/kubernetes-sigs/kind/releases)
- Docker

## Create a cluster

```
kind create cluster --name kovan-replica
```

## Create the target namespace

```
kubectl create ns kovan-replica-0-3-0
```

## Diff and Apply

```
kustomize build kovan-replica-0-3-0-kind/ | kubectl diff -f -
kustomize build kovan-replica-0-3-0-kind/ | kubectl apply -f -
```
## Watch the pods start

```
kubectl -n kovan-replica-0-3-0 get pods -w
```

### Watch the logs
```
kubectl -n kovan-replica-0-3-0 logs -f l2geth-replica-0
```
