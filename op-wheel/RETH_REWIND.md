# op-wheel rewind-reth

`op-wheel engine rewind-reth offline` performs an offline rewind of a stopped
reth node by shelling out to `reth stage unwind to-block <N>`. The companion
`engine reth-head` / `engine reth-state` commands wrap `reth db
stage-checkpoints get` and `reth db state` for inspecting the datadir offline.

Unit tests in `op-wheel/engine/reth_test.go` cover argument construction and
subprocess exit-code handling, but they cannot detect drift in reth's CLI
surface. The recipe below is the manual validation against a real reth binary.

## Prerequisites

- Rust toolchain (for building reth)
- Go toolchain (for building op-wheel)
- `curl` and `jq`

## 1. Build reth

```bash
cd /path/to/reth
cargo build -p reth
# Binary at: target/debug/reth
```

## 2. Build op-wheel

```bash
cd /path/to/optimism
go build -o op-wheel ./op-wheel/cmd
# Binary at: ./op-wheel
```

## 3. Start reth in dev mode

```bash
DATADIR=$(mktemp -d)
echo "Using datadir: $DATADIR"

reth node --dev \
  --dev.block-time 1s \
  --datadir "$DATADIR" \
  --http \
  --http.api all &

RETH_PID=$!
echo "reth PID: $RETH_PID"
```

Wait for RPC to be ready:

```bash
until curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq -r '.result' > /dev/null 2>&1; do
  sleep 1
done
echo "RPC ready"
```

## 4. Wait for blocks to be produced

Wait until the chain has enough blocks (e.g., at least 20):

```bash
while true; do
  HEX=$(curl -s -X POST http://localhost:8545 \
    -H "Content-Type: application/json" \
    -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq -r '.result')
  BLOCK=$((HEX))
  echo "Current block: $BLOCK"
  if [ "$BLOCK" -ge 20 ]; then
    break
  fi
  sleep 2
done
```

Record the head before rewinding:

```bash
HEAD_BEFORE=$BLOCK
REWIND_TO=10
echo "Head before rewind: $HEAD_BEFORE"
echo "Will rewind to: $REWIND_TO"
```

## 5. Stop reth

```bash
kill $RETH_PID
wait $RETH_PID 2>/dev/null
echo "reth stopped"
```

## 6. Run op-wheel rewind-reth offline

```bash
./op-wheel engine rewind-reth offline \
  --to $REWIND_TO \
  --reth-binary $(which reth || echo ./target/debug/reth) \
  --reth-datadir "$DATADIR" \
  --reth-chain dev

echo "Exit code: $?"
```

Expected output:
- Log line: `Executing reth stage unwind ...` with args `[stage unwind --datadir ... --chain dev to-block 10]`
- Log line: `Successfully rewound reth to block ...`
- Exit code: `0`

## 7. Restart reth and verify

```bash
reth node --dev \
  --dev.block-time 1s \
  --datadir "$DATADIR" \
  --http \
  --http.api all &

RETH_PID=$!

# Wait for RPC
until curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq -r '.result' > /dev/null 2>&1; do
  sleep 1
done

# Check head
HEX=$(curl -s -X POST http://localhost:8545 \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' | jq -r '.result')
HEAD_AFTER=$((HEX))
echo "Head after rewind: $HEAD_AFTER"
```

Verify:
- `HEAD_AFTER` should be `$REWIND_TO` (10) immediately on startup, before new blocks are mined.
  Note: dev mode resumes mining, so the head will increase quickly — check immediately.

## 8. Cleanup

```bash
kill $RETH_PID 2>/dev/null
rm -rf "$DATADIR"
```

## What success looks like

```
Head before rewind: 25
Will rewind to: 10
reth stopped
Executing reth stage unwind ...
Successfully rewound reth to block 10
Exit code: 0
Head after rewind: 10
```

## Troubleshooting

| Issue | Cause | Fix |
|---|---|---|
| `reth binary not found` | Wrong `--reth-binary` path | Use absolute path to reth binary |
| `reth stage unwind failed with exit code 1` | Database locked or corrupted | Make sure reth is fully stopped before rewinding |
| Head after rewind is higher than expected | Dev mode resumed mining before you checked | Query `eth_blockNumber` immediately, or start without `--dev.block-time` |
| `No such file or directory` for datadir | Wrong `--reth-datadir` | Check the path printed in step 3 |
