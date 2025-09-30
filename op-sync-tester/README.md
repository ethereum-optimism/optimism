# op-sync-tester

`op-sync-tester` mocks EL layer to test CL sync behavior. It proxies a real execution layer RPC node and controls block visibility to provide a controlled testing environment for consensus layer clients.

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

The op-sync-tester sits between CL client and real EL node, intercepts RPC calls and validates them against real EL data.

```
CL Client (op-node) --engine/eth APIs--> op-sync-tester --proxied RPC--> Real EL Node (op-geth)
```

Each test session is identified by UUID v4 in the URL path. Query parameters set the initial block state (latest/safe/finalized).

URL format: `http://host:port/chain/{chain_id}/synctest/{uuid}?latest=N&safe=M&finalized=K`

The service filters block queries based on session state - blocks beyond `latest` return not found, simulating a partially synced chain. When `el_sync_target` param is set, NewPayload can jump to that block to simulate EL Sync.

### `sync` RPC namespace

- `sync_getSession`
- `sync_deleteSession`
- `sync_resetSession`
- `sync_listSessions`

### `engine` RPC namespace

- ForkchoiceUpdated V1/V2/V3
- NewPayload V1/V2/V3/V4
- GetPayload V1/V2/V3/V4

Validates payloads against real EL data. Returns `VALID`, `SYNCING`, or `INVALID`.

### `eth` RPC namespace

- `eth_chainId`
- `eth_getBlockByNumber`
- `eth_getBlockByHash`
- `eth_getBlockReceipts`

## Configuration

See `--help` for full flag list. Common ones:

- `--config` - config file (default: config.yaml)
- `--rpc.addr` / `--rpc.port` - RPC server address
- `--log.level` - log level
- `--metrics.enabled` / `--metrics.port` - metrics

Environment variables with `OP_SYNC_TESTER_` prefix work too.

Config file format:
```yaml
synctesters:
  <name>:
    chain_id: <chain_id>
    el_rpc: <rpc_url>
```

See `example_config.yaml`.

## Design notes

Read-only mode - only reads from real EL, doesn't modify state.
Verifier mode only - sequencer mode not supported yet (NoTxPool must be true).
Each session maintains independent state.

## Testing

```bash
go test ./...
```

Integration tests in [`op-e2e`](../op-e2e/).
