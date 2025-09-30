# OP Supernode

**OP Supernode** is a higher order layer for running multiple OP Stack chains within a single process. It provides chain containerization and virtualization capabilities that enable efficient multi-chain operation with isolated execution environments.

## Overview

The OP Supernode transforms the traditional single-chain OP Node architecture into a multi-tenant system where multiple chains can run simultaneously as lightweight, in-memory workers. Each chain operates in its own virtualized space with isolated configuration, storage, and logging.

## Key Capabilities

### 🔗 Chain Containerization

Chains run as in-memory workers managed by **Chain Containers**. Each container:
- Manages the complete lifecycle of a Virtual OP Node (start, stop, pause, resume)
- Provides automatic restart and error recovery
- Runs in isolated goroutines with controlled concurrency

**Architecture:**
```
Supernode
  ├── ChainContainer (Chain 901)
  │     └── VirtualNode (OP Node instance)
  ├── ChainContainer (Chain 902)
  │     └── VirtualNode (OP Node instance)
  └── ChainContainer (Chain 903)
        └── VirtualNode (OP Node instance)
```

### ⚙️ Virtualized CLI Configuration

All OP Node configuration flags can be applied per-chain using the `-vn.*` flag prefix system:

- **`-vn.all.<flag>`** - Apply to all configured chains
- **`-vn.<chainID>.<flag>`** - Apply to a specific chain only

The Supernode intercepts these flags before CLI parsing and dynamically generates per-chain configuration objects. This allows each Virtual Node to have its own:
- Rollup configuration
- Sync modes and parameters
- Network settings

**Example:**
```bash
op-supernode \
  --chains 901,902,903 \
  -vn.all.l1=http://l1-endpoint:8545 \
  -vn.901.l2=http://chain-901:8551 \
  -vn.902.l2=http://chain-902:8551 \
  -vn.903.l2=http://chain-903:8551 \
```

### 💾 Virtualized DataDir

Each chain's persistent data is isolated within the Supernode's data directory:

```
datadir/
  ├── 901/
  │   └── safe_db/          # Safe head database for chain 901
  ├── 902/
  │   └── safe_db/          # Safe head database for chain 902
  └── 903/
      └── safe_db/          # Safe head database for chain 903
```

The `safe_db` (leveldb) is automatically scoped to `<datadir>/<chainID>/safe_db/`, preventing conflicts and enabling clean separation of chain state.

### 📝 Virtualized Logs

All log messages are epanded to include context fields:

- **`chain_id`** - The L2 chain ID the log originated from
- **`vn_id`** - An ephemeral 4-character UUID identifying the Virtual Node instance

This enables:
- Easy filtering in production logging systems (Grafana, Datadog, etc.)
- Debugging specific chain instances
- Tracking Virtual Node restarts and lifecycle events

### 🔌 Shared Resources

The Supernode optimizes resource usage by sharing L1 connections across all Virtual Nodes:

#### Shared L1 Client
- **Single TCP connection** to the L1 RPC endpoint for all chains
- **Shared cache** for L1 blocks, receipts, and transactions
- **Shared rate limiting** across all Virtual Node requests
- **Protected from closure** - Virtual Nodes cannot close the shared connection

#### Implementation
The Supernode creates L1 and L1 Beacon clients on startup and wraps them with **non-closeable wrappers** in the `resources/` package. These wrappers:
- Implement the same interfaces as the standard clients (`node.L1Client`, `node.BeaconClient`)
- Delegate all methods to the underlying client
- Override `Close()` to be a no-op, preventing accidental closure
- Are passed to Virtual Nodes via `op-node`'s `InitOverload` mechanism

## Architecture

### Components

#### Supernode
The top-level orchestrator that:
- Manages the lifecycle of all chain containers
- (Future) Handles Higher Order Validatin Activities
- Handles graceful shutdown across all chains
- Coordinates startup and initialization

#### ChainContainer
Per-chain lifecycle manager that:
- Wraps a Virtual Node with control logic
- Manages config overrides
- Passes shared resources to Virtual Nodes
- Handles pause/resume operations
- Provides restart-on-error behavior

#### VirtualNode
In-memory OP Node instance that:
- Wraps the standard OP Node implementation
- Receives virtualized configuration
- Operates with isolated resources
- Generates chain-specific logs

## Usage

### Building

```bash
just op-supernode
```

### Running

**Basic usage with multiple chains:**

```bash
./bin/op-supernode \
  --sample "example" \
  --chains 901,902 \
  --data-dir ./supernode-data \
  --l1 http://localhost:8545 \
  --l1.beacon http://localhost:5052 \
  -vn.901.l2=http://localhost:9001 \
  -vn.901.l2.jwt-secret=./jwt-901.txt \
  -vn.901.rollup.config=./rollup-901.json \
  -vn.902.l2=http://localhost:9002 \
  -vn.902.l2.jwt-secret=./jwt-902.txt \
  -vn.902.rollup.config=./rollup-902.json
```

**Using environment variables:**

```bash
export OP_SUPERNODE_CHAINS=901,902,903
export OP_SUPERNODE_SAMPLE="production"
export OP_SUPERNODE_DATA_DIR=/var/lib/supernode
export OP_SUPERNODE_L1_ETH_RPC=$L1_RPC
export OP_SUPERNODE_L1_BEACON=$L1_BEACON

./bin/op-supernode \
  -vn.901.l2=$CHAIN_901_RPC \
  -vn.902.l2=$CHAIN_902_RPC \
  -vn.903.l2=$CHAIN_903_RPC
```

### Configuration

#### Required Flags

- `--sample` - Sample configuration string (required for now)
- `--chains` - Comma-separated list of chain IDs to run
- `--l1` - L1 RPC endpoint (shared across all chains)

#### Optional Flags

- `--l1.beacon` - L1 Beacon endpoint (shared across all chains)
- `--data-dir` - Root data directory for all chains (default: `./datadir`)
- Standard OP Service flags for logging, metrics, pprof, and RPC

#### Virtual Node Flags

All standard OP Node flags can be prefixed with:
- `-vn.all.<flag>` - Applies to all chains
- `-vn.<chainID>.<flag>` - Applies to specific chain

**Common patterns:**

```bash
# Shared L1 for all chains (supernode-level flags)
--l1 http://l1:8545
--l1.beacon http://l1:5052

# Per-chain L2 execution engines
-vn.901.l2=http://op-geth-901:8551
-vn.902.l2=http://op-geth-902:8551

# Per-chain rollup configs
-vn.901.rollup.config=./rollup-901.json
-vn.902.rollup.config=./rollup-902.json

# Shared sync configuration (applies to all virtual nodes)
-vn.all.syncmode=execution-layer
```

## Current Limitations

- **P2P is disabled** for Virtual Nodes (can be enabled later for unsafe head sync)
- **No RPC server** exposed per-chain (coming soon)
- Minimal chain container lifecycle controls (pause/resume implemented but not exposed via API)
- No Metrics enabled on Virtual Nodes yet. Once a Rerverse Proxy is implemented, this and RPC service will be possible.
