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
el_rpc: https://localhost:8545
```

Run the service:
```bash
go run ./cmd --config=config.yaml --rpc.addr=127.0.0.1 --rpc.port=9000
```

Initialize test session
```bash
cast rpc --rpc-url='http://localhost:8545/synctest?head=3&safe=2&finalized=1' sync_init
```

### Build docker image

Not available yet.

## Overview

### `sync` RPC namespace

#### `sync_init`

Initializes the testing session from query string:
- `head`: decimal number
- `safe`: decimal number
- `finalized`: decimal number

### `admin` RPC namespace

On the global RPC an `admin` namespace is available.

#### `admin_clearSessions`

Clears sessions.
