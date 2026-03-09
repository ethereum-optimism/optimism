# Overview

Generating the first SuperRoot used for an Interop migration

## Prerequisites:

1. git clone or pull the latest develop branch of the optimism repo
2. Go Installed
    - You can follow the instructions in the [CONTRIBUTING.md](http://CONTRIBUTING.md) to install all software dependencies of the repo
3. A trusted SuperNode RPC endpoint.

## Generation:

A SuperRoot with this script can be fetched at a provided timestamp, or if none is provided the latest finalized timestamp reported by SuperNode will be used.

```bash
go run op-chain-ops/cmd/check-super-root/main.go --super-node-rpc $SUPER_NODE_RPC
```

Output:

```bash
INFO [05-19|16:24:50.736] Super root fetched successfully superRoot=0xa47cfdd734e7db568eea0531c5fa6117398a5218d76371a17656e13836a8a44f timestamp=1747682546 chains=2
```

Record the bytes32 hash of the super root and the timestamp the SuperRoot is valid for

## Environment Variables

Alternatively, you can use environment variables to configure the script:

- `CHECK_SUPER_ROOT_SUPER_NODE_RPC`: SuperNode RPC endpoint.
- `CHECK_SUPER_ROOT_TIMESTAMP`: Target timestamp for super root calculation.

# Validation Checks

A SuperRoot can be validated with the script and providing the timestamp the SuperRoot was created for:

```bash
go run op-chain-ops/cmd/check-super-root/main.go --super-node-rpc $SUPER_NODE_RPC --timestamp $SUPER_ROOT_TIMESTAMP
```

The output should match the above

```bash
INFO [05-19|16:24:50.736] Super root fetched successfully superRoot=0xa47cfdd734e7db568eea0531c5fa6117398a5218d76371a17656e13836a8a44f timestamp=1747682546 chains=2
```
