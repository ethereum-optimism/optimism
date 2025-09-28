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
    engine_kind: geth  # Optional: geth, reth, or erigon (defaults to geth)
    sync_mode: full    # Optional: full, snap, etc. (defaults to network-specific)
    network_type: sepolia  # Optional: mainnet, sepolia, goerli, etc.
```

### Engine Implementation Mocking

op-sync-tester now supports mocking different EL (Execution Layer) implementations to test L2CL (L2 Consensus Layer) behavior with various engine types:

- **Geth**: Conservative sync approach, does not support post-finalization EL sync
- **Reth**: Aggressive sync with better performance, supports post-finalization EL sync
- **Erigon**: Optimized for archival data, supports post-finalization EL sync

Each engine implementation can be configured with different sync modes and network characteristics to simulate real-world scenarios.

Example with multiple engine types:
```yaml
synctesters:
  sepolia-geth:
    chain_id: 11155420
    el_rpc: https://sepolia.optimism.io
    engine_kind: geth
    sync_mode: full
    network_type: sepolia

  sepolia-reth:
    chain_id: 11155420
    el_rpc: https://sepolia.optimism.io
    engine_kind: reth
    sync_mode: snap
    network_type: sepolia

  mainnet-with-regenesis:
    chain_id: 10
    el_rpc: https://mainnet.optimism.io
    engine_kind: erigon
    sync_mode: full
    network_type: mainnet  # Supports regenesis
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
