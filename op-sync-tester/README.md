# op-sync-tester

## Usage

### Build from source

```bash
# from op-sync-tester dir:
just op-sync-tester
./bin/op-sync-tester --help
```

### Run from source

Example config:
```yaml
synctesters:
  sepolia:
    chain_id: 11155420
    el_rpc:  https://sepolia.optimism.io
```

Run the service:
```bash
go run ./cmd --config=config.yaml --rpc.addr=127.0.0.1 --rpc.port=9000
```

Initialize test session
```bash
cast rpc --rpc-url='http://localhost:9000/chain/11155420/synctest/41a16f5c-24a9-4a6a-b072-917d55ca5d39?latest=3&safe=2&finalized=1' eth_chainId
"11155420"
```

### Build docker image

Not available yet.

## Overview

### `sync` RPC namespace

### `engine` RPC namespace

### `eth` RPC namespace
