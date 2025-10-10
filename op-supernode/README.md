## OP Supernode

Run multiple OP Stack chains in a single process. OP Supernode virtualizes OP Node so each chain runs as an isolated in-memory worker with per-chain config, data, and logs.

### Highlights
- Multi-chain in one binary; lightweight per-chain workers
- Per-chain config via `-vn.*` flag prefixing
- Isolated data directories per chain
- Structured logs with `chain_id` and `vn_id`
- Shared L1 RPC and Beacon clients with non-closeable wrappers

### How it works
```
Supernode: Runs Containers
  ├── ChainContainer (901) Manages:
      ├── VirtualNode ── In Memory OP Node (901)
      └── (FUTURE) Engine Controller (901)
  ├── ChainContainer (902) Manages:
      ├── VirtualNode ── In Memory OP Node (902)
      └── (FUTURE) Engine Controller (902)
  └── ChainContainer (903) Manages:
      ├── VirtualNode ── In Memory OP Node (903)
      └── (FUTURE) Engine Controller (903)
```
- Supernode orchestrates chain containers (start/stop/restart, pause/resume, shutdown)
- ChainContainer applies per-chain config and passes shared resources
- VirtualNode runs OP Node with isolated resources and context-rich logging

### Quickstart
Build:
```bash
just op-supernode
```

Run multiple chains:
```bash
./bin/op-supernode \
  --chains 901,902 \
  --data-dir ./supernode-data \
  --l1 http://localhost:8545 \
  --l1.beacon http://localhost:5052 \
  -vn.901.l2=http://localhost:9001 \
  -vn.901.rollup.config=./rollup-901.json \
  -vn.902.l2=http://localhost:9002 \
  -vn.902.rollup.config=./rollup-902.json \
  -vn.all.l2.jwt-secret=./jwt-902.txt
```

Environment variables:
```bash
export OP_SUPERNODE_CHAINS=901,902,903
export OP_SUPERNODE_DATA_DIR=/var/lib/supernode
export OP_SUPERNODE_L1_ETH_RPC=$L1_RPC
export OP_SUPERNODE_L1_BEACON=$L1_BEACON

./bin/op-supernode \
  -vn.901.l2=$CHAIN_901_RPC \
  -vn.902.l2=$CHAIN_902_RPC \
  -vn.903.l2=$CHAIN_903_RPC
```

### Configuration
- Required: `--chains`, `--l1`
- Optional: `--l1.beacon`, `--data-dir` (default `./datadir`), standard op-service flags (logging, metrics, pprof, RPC)

Per-chain flags are prefixed:
- `-vn.all.<flag>` applies to all chains
- `-vn.<chainID>.<flag>` applies to one chain

Common examples:
```bash
# Supernode-level L1
--l1 http://l1:8545
--l1.beacon http://l1:5052

# Per-chain L2 execution engines
-vn.901.l2=http://op-geth-901:8551
-vn.902.l2=http://op-geth-902:8551

# Per-chain rollup configs
-vn.901.rollup.config=./rollup-901.json
-vn.902.rollup.config=./rollup-902.json

# Apply to all chains
-vn.all.syncmode=execution-layer
```

### Data and logs
Data layout:
```
<data-dir>/
  ├── 901/
  │   └── safe_db/
  ├── 902/
  │   └── safe_db/
  └── 903/
      └── safe_db/
```

Logs include `chain_id` and a short-lived `vn_id` for filtering and debugging.

### RPC Routing
RPC Clients are created in namespaced paths, so the supernode has a single RPC URL which acts as an RPC router to the virtual nodes
```
/
  ├── 901/
  ├── 902/
  └── 903/
```
calling RPC methods on `/901` will route the method to the Virtual Node for that chain.

### Limitations
- P2P disabled for Virtual Nodes (unsafe head sync possible later)
- Pause/resume exists but not yet exposed via API
- Virtual Node metrics are untested and expected non-functional currently
