# Migrating op-geth State to op-reth

This documents how to bootstrap an op-reth node from an existing op-geth node's state. This is needed because Boba chains migrated to Bedrock at a specific block (the "migration block"), and op-reth needs the state at that block to start syncing.

Pre-built snapshots are available on the [snapshot downloads](../../boba-docs/dev-docs/node-operators/5_snapshot-downloads.md) page. If you use a pre-built snapshot, you can skip directly to running the node — the `init-state` step has already been done for you.

The published migration artifacts (state dump JSONL and block header RLP) are also available on the snapshot downloads page. You can use these to independently verify or reproduce the initial reth database without needing a synced geth/erigon node.

This guide is for operators who want to regenerate the database from scratch, or understand how the published snapshots were produced.

## Overview

1. Extract the full state from op-geth at the migration block
2. Convert the state dump to op-reth's JSONL format
3. Extract the migration block header as RLP
4. Initialize op-reth with the state dump
5. Start op-reth + op-node

## Prerequisites

- A running op-geth node synced to at least the migration block, with `debug` API enabled
- `op-reth` binary (built from `optimism/rust/` workspace: `cargo build --bin op-reth --release`)
- `jq`, `curl`, `base64`, `od` available in PATH
- The migration block's header in RLP format (see Step 2)

## Chain-Specific Parameters

| Parameter | Boba Sepolia | Boba Mainnet |
|-----------|-------------|--------------|
| L2 Chain ID | 28882 | 288 |
| Migration block | 511 | 1149019 |
| L1 Chain | Sepolia (11155111) | Ethereum (1) |
| op-reth `--chain` | `boba-sepolia` | `boba` |

## Step 1: Dump all accounts from op-geth

**Important**: `debug_dumpBlock` only returns 256 accounts per page and does not paginate. For blocks with more than 256 accounts, you MUST use `debug_accountRange` with pagination. The included `dump-all-accounts.sh` handles this automatically.

```bash
# Set the geth RPC endpoint (must have debug API enabled)
export RPC_URL=http://localhost:8545

# For Boba Sepolia (migration block 511 = 0x1ff):
./dump-all-accounts.sh 0x1ff state-dump-511-full.json

# For Boba Mainnet (migration block 1149019 = 0x118a5b):
./dump-all-accounts.sh 0x118a5b state-dump-1149019-full.json
```

If dumping from an erigon node instead, use `dump-all-accounts-erigon.sh` (erigon uses slightly different RPC parameters and base64 cursors).

This produces a JSON file with the structure:
```json
{
  "result": {
    "root": "<state-root-hex-no-0x>",
    "accounts": {
      "<address>": {
        "balance": "<decimal>",
        "nonce": <number>,
        "code": "0x...",
        "storage": { "<key>": "<value-hex-no-0x>", ... },
        "address": "0x..."
      }
    }
  }
}
```

Verify the account count is reasonable. Boba Sepolia block 511 has 2239 accounts. Boba Mainnet will have significantly more.

## Step 2: Convert to op-reth JSONL format

```bash
# For Boba Sepolia:
./convert-to-reth.sh state-dump-511-full.json reth-state-511.jsonl

# For Boba Mainnet:
./convert-to-reth.sh state-dump-1149019-full.json reth-state-1149019.jsonl
```

The JSONL format:
- Line 1: `{"root":"0x<state-root-hash>"}`
- Lines 2+: One account per line with `address`, `balance`, `nonce`, `code`, `storage`

Key format differences from geth:
- **State root**: needs `0x` prefix (geth omits it)
- **Storage values**: need `0x` prefix and zero-padded to 64 hex chars
- **Balance/nonce**: decimal strings (op-reth accepts both decimal and hex)

## Step 3: Extract the migration block header

The block header must be RLP-encoded. Extract it from geth:

```bash
# Get the block header as JSON, then RLP-encode it.
# The easiest method is to use debug_getRawHeader:
BLOCK_HEX=0x1ff  # or 0x118a5b for mainnet

curl -s -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"debug_getRawHeader\",\"params\":[\"$BLOCK_HEX\"],\"id\":1}" \
  | jq -r '.result' > header.rlp
```

If `debug_getRawHeader` is not available, you can use `debug_getRawBlock` and strip the block body (transactions/uncles), though this requires more work.

## Step 4: Initialize op-reth

The chain spec is supplied via the JSON files in [`../../chainspecs/`](../../chainspecs/) (Boba is no longer built into upstream op-reth — see that directory's README for context).

```bash
# Generate a JWT secret for Engine API authentication
openssl rand -hex 32 > jwt.hex

# For Boba Sepolia:
op-reth init-state \
  --chain=../../chainspecs/boba-sepolia.json \
  --datadir=./reth-data \
  --without-ovm \
  --header=header-511.rlp \
  reth-state-511.jsonl

# For Boba Mainnet:
op-reth init-state \
  --chain=../../chainspecs/boba.json \
  --datadir=./reth-data \
  --without-ovm \
  --header=header-1149019.rlp \
  reth-state-1149019.jsonl
```

The `--without-ovm` flag tells op-reth to:
1. Create a dummy chain from block 0 to (migration_block - 1)
2. Append the migration block header
3. Import the state and verify the state root matches

**If the state root doesn't match**, the most likely cause is an incomplete account dump (see the `debug_dumpBlock` pagination issue in Step 1).

## Step 5: Start op-reth

```bash
op-reth node \
  --chain=../../chainspecs/boba-sepolia.json \
  --datadir=./reth-data \
  --http --http.port=8545 \
  --authrpc.port=8551 \
  --authrpc.jwtsecret=jwt.hex \
  --http.api=eth,net,web3,debug,txpool \
  --ws --ws.port=8546
```

## Step 6: Start op-node

op-node drives op-reth via the Engine API, deriving L2 blocks from L1 data.

```bash
op-node \
  --l1=<L1_RPC_URL> \
  --l1.beacon=<L1_BEACON_URL> \
  --l2=http://localhost:8551 \
  --l2.jwt-secret=jwt.hex \
  --network=boba-sepolia \
  --rpc.addr=0.0.0.0 \
  --rpc.port=9545
```

For mainnet, use `--network=boba-mainnet` (or `--rollup.config=<path>` with a rollup config JSON).

## Step 7: Verify

```bash
# Check that the block number advances past the migration block
curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

# Confirm via Engine API that forkchoice is valid (requires JWT auth)
# op-node will do this automatically during startup
```

## Troubleshooting

### State root mismatch during init-state

The most common cause is `debug_dumpBlock` returning only 256 accounts (its default page size). Always use `dump-all-accounts.sh` which paginates via `debug_accountRange`.

To verify: compare `jq '.result.accounts | length'` between your dump and the actual account count. You can check the account count with:
```bash
curl -s -X POST "$RPC_URL" \
  -H "Content-Type: application/json" \
  -d "{\"jsonrpc\":\"2.0\",\"method\":\"debug_accountRange\",\"params\":[\"$BLOCK_HEX\",\"0x0000000000000000000000000000000000000000000000000000000000000000\",1,false,false,false],\"id\":1}" \
  | jq '.result'
# Check the response — if 'next' is non-null, there are more accounts
```

### op-node stuck in "waiting for EL sync"

This can happen if the Engine API forkchoice update returns SYNCING instead of VALID. Verify:
1. op-reth is running and responsive on the auth port
2. The JWT secret matches between op-node and op-reth
3. op-reth was initialized with the correct chain and state

### Mainnet considerations

Boba Mainnet (block 1149019) will have many more accounts than Sepolia (block 511). The `dump-all-accounts.sh` script may need to run for longer. The `convert-to-reth.sh` script and `init-state` will also take longer due to the larger state. The `--without-ovm` flag generates ~1.1M dummy blocks for mainnet (vs 511 for sepolia), which takes roughly 90 seconds.
