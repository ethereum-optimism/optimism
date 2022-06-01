# opnode

This is the reference implementation of the [rollup-node spec](../specs/rollup-node.md).

## Compiling

From the repository root:

```shell
go build -o op ./op-node/cmd
go test ./op-node/...
```

## Running

Options can be reviewed with:

```shell
./op --help
```

To start syncing the rollup:

Connect to at least one L1 RPC and L2 execution engine:

- L1: use any L1 node / RPC (websocket connection path may differ)
- L2: run the Optimism fork of geth: <https://github.com/ethereum-optimism/reference-optimistic-geth>

Initialize the L2 chain with a `genesis.json` chain spec like L1, with the Merge fork activated from genesis.

Specify genesis details:

- L1 number / hash: starting-point of L2 chain inputs
- L2 genesis hash: to confirm we are building on the correct L2 genesis

```shell
# websockets or IPC preferred for event notifications to improve sync, http RPC works with adaptive polling.
op \
  --l1=ws://localhost:8546 --l2=ws//localhost:9001 \
  --genesis.l1-num=.... --genesis.l1-hash=..... --genesis.l2-hash=....
```
