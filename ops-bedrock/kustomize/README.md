# kustomize

Contains Kustomize configs for deploying Bedrock replicas.

## Architecture

Kustomize configurations for individual networks extend an overlay names after the network. All configurations will
extend the `node` component, which configures `op-node` and `op-geth`.

The [beta-1-example](./beta-1-example) directory contains a working example that connects to the beta devnet.

Before running this yourself, make sure to replace the `OP_NODE_L1_ETH_RPC` environment variable
in `components/node/op-node/config.yaml` with an actual L1 RPC, and the `jwt-secret.txt` and `p2p-node-key.txt` files with a 64-character random hex string.