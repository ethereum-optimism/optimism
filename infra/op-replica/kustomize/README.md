# Kustomize kubernetes deployments

[Kustomize](https://kustomize.io/) is a way to build custom kubernetes manifests in a template-free way.

This directory describes how to build and deploy Optimistic Ethereum software to a kubernetes cluster.

## Prerequisits

- `kubectl` **Minimum version v1.20** [Install notes](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/#install-kubectl-on-linux)
- a kubernetes cluster

## Structure

The kustomization starts from a folder in `./bases`. Directories under `bases` should describe components in their most generic form. It is possible to deploy directly from a `bases` directory if no resource modification is needed.

The other folders are "overlays", or configuration that will modify resources defined in `bases`.

## Kustomizing with a new overlay

Kustomize build resources based on `kustomization.yaml` files.

To create a new overlay, create a new directory with a new `kustomization.yaml` file that refers to the base resources to target.

Add any global modifers, the base resources to target and any modifications. **A valid base target is any directory with a `kustomization.yaml` file**.

See a detailed description of a kovan replica [here](./README-kovan-replica.md)

## Building

`kustomize` can be run in 2 different way, with the built in `kubectl` flag or using the stand alone binary. **The binary should be used for this repository** as the `kubectl` releases are lagging on the `envs` feature.

Using kubectl with the `-k` flag and a target directory will build and diff or apply the generated manifests.

```
kubectl diff -k ./bases/configmaps/
kubectl apply -k ./bases/configmaps/
```

Using a `kustomize` binary release, we simplay build the manifests and pipe them to `kubectl` on STDIN.

```
kustomize build ./bases/configmaps/ | kubectl diff -f -
kustomize build ./bases/configmaps/ | kubectl apply -f -
```

## Example deployment

See [./README-kovan-replica-deploy.md](README-kovan-replica-deploy.md)
