<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [op-node](#op-node)
  - [Compiling](#compiling)
  - [Testing](#testing)
  - [Running](#running)
  - [L2 Genesis Generation](#l2-genesis-generation)
  - [L1 Devnet Genesis Generation](#l1-devnet-genesis-generation)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# op-node

This is the reference implementation of the [rollup-node spec](https://github.com/ethereum-optimism/specs/blob/main/specs/protocol/rollup-node.md).
It can be thought of like the consensus layer client of an OP Stack chain where it must run with an OP Stack execution layer client
like [op-geth](https://github.com/ethereum-optimism/op-geth).

## Compiling

Compile a binary:

```shell
make op-node
```

## Testing

Run op-node unit tests:

```shell
make test
```

## Running

Configuration options can be reviewed with:

```shell
./bin/op-node --help
```

[eth-json-rpc-spec]: https://ethereum.github.io/execution-apis/api-documentation

To start syncing the rollup:

Connect to one L1 Execution Client that supports the [Ethereum JSON-RPC spec][eth-json-rpc-spec],
an L1 Consensus Client that supports the [Beacon Node API](https://ethereum.github.io/beacon-APIs) and
an OP Stack based Execution Client that supports the [Ethereum JSON-RPC spec][eth-json-rpc-spec]:

- L1: use any L1 client, RPC, websocket, or IPC (connection config may differ)
- L2: use any OP Stack Execution Client like [`op-geth`](https://github.com/ethereum-optimism/op-geth)

Note that websockets or IPC is preferred for event notifications to improve sync, http RPC works with adaptive polling.

```shell
./bin/op-node \
  --l1=ws://localhost:8546 \
  --l1.beacon=http://localhost:4000 \
  --l2=ws://localhost:9001 \
  --rollup.config=./path-to-network-config/rollup.json \
  --rpc.addr=127.0.0.1 \
  --rpc.port=7000
```

## L2 Genesis Generation

The `op-node` can generate geth compatible `genesis.json` files. These files
can be used with `geth init` to initialize the `StateDB` with accounts, storage,
code and balances. The L2 state must be initialized with predeploy contracts
that exist in the state and act as system level contracts. The `op-node` can
generate a genesis file with these predeploys configured correctly given
an L1 RPC URL, a deploy config, L2 genesis allocs and a L1 deployments artifact.

The deploy config contains all of the config required to deploy the
system. Examples can be found in `packages/contracts-bedrock/deploy-config`. Each
deploy config file is a JSON file. The L2 allocs can be generated using a forge script
in the `contracts-bedrock` package and the L1 deployments are a JSON file that is the
output of doing a L1 contracts deployment.

Example usage:

```bash
$ ./bin/op-node genesis l2 \
  --l1-rpc $ETH_RPC_URL \
  --deploy-config <PATH_TO_MY_DEPLOY_CONFIG> \
  --l2-allocs <PATH_TO_L2_ALLOCS> \
  --l1-deployments <PATH_TO_L1_DEPLOY_ARTIFACT> \
  --outfile.l2 <PATH_TO_WRITE_L2_GENESIS> \
  --outfile.rollup <PATH_TO_WRITE_OP_NODE_CONFIG>
```

## L1 Devnet Genesis Generation

It is also possible to generate a devnet L1 `genesis.json` file. The L1 allocs can
be generated with the foundry L1 contracts deployment script if the extra parameter
`--sig 'runWithStateDump()` is added to the deployment command.

```bash
$ ./bin/op-node genesis l1 \
   --deploy-config $CONTRACTS_BEDROCK/deploy-config \
   --l1-deployments <PATH_TO_L1_DEPLOY_ARTIFACT> \
   --l1-allocs <PATH_TO_L1_ALLOCS>
```
