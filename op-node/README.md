# op-node

This is the reference implementation of the [rollup-node spec](../specs/rollup-node.md).

## Compiling

Compile a binary:
```shell
cd op-node
go build -o bin/op-node ./cmd
```

## Testing

Run op-node unit tests:
```shell
cd op-node
go test ./...
```

Run end-to-end tests:
```shell
cd op-e2e
go test ./...
```

## Running

Options can be reviewed with:

```shell
./bin/op-node --help
```

To start syncing the rollup:

Connect to at least one L1 RPC and L2 execution engine:

- L1: use any L1 node / RPC (websocket connection path may differ)
- L2: run the Optimism fork of geth: [`op-geth`](https://github.com/ethereum-optimism/op-geth)

```shell
# websockets or IPC preferred for event notifications to improve sync, http RPC works with adaptive polling.
op \
  --l1=ws://localhost:8546 --l2=ws//localhost:9001 \
  --rollup.config=./path-to-network-config/rollup.json \
  --rpc.addr=127.0.0.1 \
  --rpc.port=7000
```
