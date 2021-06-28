# RPC-Proxy

## rpc-proxy settings

There are only 2 settings currently surfaced through the container.
1. What methods to allow
2. The address of the upstream sequencer

Both are configured through environmental variables.
```
- name: SEQUENCER
  value: sequencer:8545
- name: RPC_METHODS_ALLOWED
  value: eth_blockNumber,eth_getBlockByNumber,eth_getBlockRange,eth_sendRawTransaction
```

## Deploy the rpc-proxy

These deployments use `kustomize` which references base resources and allows overlays for modification.

The overlay directories contain the deviations from the `bases` collection of default resources.

Target the base directory to deploy the base configuration.
```
kubectl diff -k ops/infra/rpc-proxy/bases
kubectl apply -k ops/infra/rpc-proxy/bases
```

Target an overlay directory to apply environmental specific modifications.
```
kubectl diff -k ops/infra/rpc-proxy/goerli-devnet/
kubectl apply -k ops/infra/rpc-proxy/goerli-devnet/
```

## Setup once for GCP ingress

Create a ip reservation on GCP with a useful name
```
gcloud compute addresses create goerli-sequencer --global
gcloud compute addresses create kovan-sequencer --global
gcloud compute addresses create mainnet-sequencer --global
```

View the assigned IP address
```
$ gcloud compute addresses list
NAME                   ADDRESS/RANGE   TYPE      PURPOSE  NETWORK  REGION  SUBNET  STATUS
goerli-sequencer       35.190.76.113   EXTERNAL                                    IN_USE
```

Update Cloudflare dashboard with the address and corresponding hostname.

## Ingress and ManagedCertificates

This deployment uses the GKE Ingress to avoid routing through another pod.

The offical documentation can be found [here](https://cloud.google.com/kubernetes-engine/docs/how-to/managed-certs)

It's **very important that `readinessProbe` passes** for the pods backing a GKE Ingress, otherwise it will be marked down.
