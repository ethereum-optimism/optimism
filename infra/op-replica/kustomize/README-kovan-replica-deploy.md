# kovan-replica-deploy

## Prerequisites

- `kubectl` **Minimum version v1.20** [Install notes](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/#install-kubectl-on-linux)
- `kind` from a recent [kind release](https://github.com/kubernetes-sigs/kind/releases)
- Docker

## Create a cluster

```
kind create cluster --name kovan-replica
```

## Create the target namespace

```
kubectl create ns kovan-replica
```

## Diff and Apply

```
kubectl diff -k kustomize/replica/overlays/kind-kovan-replica/
kubectl apply -k kustomize/replica/overlays/kind-kovan-replica/
```
## Watch the pods start

```
kubectl -n kovan-replica get pods -w
```

### Watch the logs
```
kubectl -n kovan-replica logs -f l2geth-replica-0
```

### Check the replica status
```
kubectl logs -f -l app=replica-healthcheck
```
