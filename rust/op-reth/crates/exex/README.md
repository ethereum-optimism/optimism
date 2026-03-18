# op-reth historical proofs
[![codecov](https://codecov.io/gh/op-rs/op-reth/branch/main/graph/badge.svg)](https://app.codecov.io/gh/op-rs/op-reth/tree/unstable/crates%2Foptimism?components%5B0%5D=op%20historical%20proof)

![Description](assets/op-rs-logo.png)
> **⚠️ Under Construction**
>
> This is a work in progress. Stay tuned!

## Motivation

Reliable access to recent historical state via `eth_getProof` is a critical requirement for rollups and L2 infrastructure built on Ethereum.

As described in Reth issue [#18070](https://github.com/paradigmxyz/reth/issues/18070), many applications on Optimism and other rollups (e.g. Base infrastructure, ENS, fault-proof systems) depend on fast and reliable `eth_getProof` queries within a bounded challenge window (typically 7 days). At present, the lack of reliable recent-state proof support is a blocker for broader Reth adoption in these environments.

The core issue lies in Reth's architecture for historical state calculation. To serve `eth_getProof` for a historical block, Reth must perform an **in-memory revert**, applying state diffs backwards from the chain tip. While efficient for recent blocks, reverting state for a block 7 days ago requires loading thousands of changesets into Memory. This operation is computationally expensive and often causes the node to crash due to **Out-Of-Memory (OOM)** errors, effectively making deep historical proofs impossible on a standard node.

While solutions like Erigon’s compressed archive format demonstrate that full historical proofs can be stored efficiently (~5 TB), most real-world use cases do not require access to *all* historical state. Instead, the overwhelming majority of applications only require proofs over a **recent, bounded time window** (e.g. the last 7 days for challenge games).

This fork introduces a **Bounded History Sidecar** architecture for historical state proofs. The goal is to provide:
- **Crash-Free Proof Generation:** Serve `eth_getProof` for deep historical blocks without the OOM risks associated with in-memory reverts.
- **Constant Storage Footprint:** Maintain a fixed storage size (linear to the configured window) rather than the unbounded growth.
- **Zero-Overhead Sync:** Utilize Reth's Execution Extensions (ExEx) to process and index history asynchronously, ensuring the main node's sync speed and tip latency are unaffected.

## Architecture: Bounded History Sidecar

This module implements a **Sidecar Storage Pattern**. Instead of burdening the main node's database with historical data, we maintain a dedicated, secondary MDBX environment optimized specifically for serving proofs.

### Core Mechanism: Versioned State
Unlike standard Reth (which stores the *current* state and calculates history by reverting diffs), this module implements a **Versioned State Store**.

1.  **`AccountTrieHistory` & `StorageTrieHistory`**: Stores the intermediate branch nodes of the Merkle Patricia Trie. Each node is versioned by block number, allowing us to traverse the exact trie structure as it existed at any past block.
2.  **`HashedAccountHistory` & `HashedStorageHistory`**: Stores the actual account data (nonce, balance) and storage slot values at the leaves of the trie, also versioned by block number.

### Initialization: State Snapshot
To ensure the service is easy to set up on existing nodes with millions of blocks, we do not require a full chain re-sync. Instead, the module requires an **Initial State Snapshot** via the CLI:

1.  **Capture:** The CLI command captures the *current* state of the blockchain (Account and Storage Tries) from the main database.
2.  **Seed:** It populates the sidecar with this baseline state.
3.  **Track:** Once initialized, the node begins tracking new blocks and maintaining history from that point forward.

This ensures that the proof window has a valid starting point immediately.

### Data Flow

1.  **Initialization:** The operator runs the initialization CLI command to snapshot the current main DB state and seed the sidecar.
2.  **Ingestion (Write):** As the node syncs, the Execution Extension (`ExEx`) captures the TrieUpdates (branch nodes) and HashedPostState (leaf values) in each block and writes them to the sidecar DB tagged with the block number.
3.  **Retrieval (Read):** When `eth_getProof` is called for a historical block, we simply look up the trie nodes valid at that specific block version.
4.  **Maintenance (Prune):** A background process monitors the chain tip. Once a block falls outside the configured window (e.g., > 7 days old), its specific history versions are deleted to reclaim space.

## New Components

### 1. `reth-optimism-exex`
This crate implements the Execution Extension (ExEx) that acts as the bridge between the main node and the sidecar storage.

- Ingestion Pipeline: Subscribes to the node's canonical state notifications to capture ExecutionOutcomes in real-time.
- Diff Extraction: Isolates the specific TrieUpdates (branch nodes) and HashedPostState (leaf values) changed in each block.
- Persistence: Writes these versioned updates to the sidecar MDBX database without blocking the main datastore.
- Lifecycle Management: Orchestrates the pruning process, ensuring the sidecar storage remains bounded by the configured window.

### 2. `reth-optimism-trie`
This crate provides the Storage Engine and Proof Logic that powers the sidecar.

- Versioned Storage: Implements MdbxProofsStorage, a specialized database schema optimized for time-series trie node retrieval.
- Proof Generation: Replaces the standard "revert-based" proof logic with a direct "lookup-based" approach.
- Pruning Logic: Implements the smart retention algorithm that safely deletes old history

### 3. RPC Overrides
The module injects custom handlers to intercept specific RPC calls:
*   **`eth_getProof`**: Checks if the requested block is historical. If so, it fetches the account and storage proofs from the secondary Proofs DB.
*   **`debug_executionWitness`**: Allows debugging and tracing against historical states.
*   **`debug_executePayload`**: Executes a payload against the historical state to generate an execution witness. 

## Hardware Requirements

Recommended specifications:

- **CPU**: 8-Core processor with good single-core performance
- **RAM**: Minimum 16 GB (32 GB recommended)
- **Storage**: NVMe SSD with adequate capacity for chain data plus snapshots
  - Calculate: `(2 × current_chain_size) + snapshot_size + 20% buffer`
  - *Note*: Storing 4 weeks of full proof history on a network like Base Testnet consumes approximately **~1 TB** of additional storage.
- **Network**: Stable internet connection with good bandwidth

## Usage

### 1. Initialization
Before starting the node with the sidecar enabled, you must initialize the proof storage. This command snapshots the current state of the main database to seed the sidecar.

```bash
op-reth proofs init \
  --datadir=path/to/reth-datadir \
  --proofs-history.storage-path=/path/to/proof-db
```

### 2. Running the Node (Syncing)

Once initialized, start the node with the --proofs-history flags to enable the sidecar service.

```bash
op-reth node \
  --chain base-sepolia \
  --datadir=/path/to/reth-datadir \
  --proofs-history \
  --proofs-history.storage-path=/path/to/proofs-db \
  --proofs-history.window=600000 \
  --proofs-history.prune-interval=15s
```

Configuration Flags

| Flag | Description | Default | Required |
| :--- | :--- | :--- | :--- |
| `--proofs-history` | Enables the historical proofs module. | `false` | No |
| `--proofs-history.storage-path` | Path to the separate MDBX database for storing proofs. | `None` | **Yes** (if enabled) |
| `--proofs-history.window` | Retention period in **blocks**. Data older than `Tip - Window` is pruned. | `1,296,000` (~30 days) | No |
| `--proofs-history.prune-interval` | How frequently the pruner runs to delete old data. | `1h` | No |

### 3. Management

We provide custom CLI commands to manage the proof history manually.

`op-reth proofs prune`
Manually triggers the pruning process. Useful for reclaiming space immediately.

```bash
op-reth proofs prune \
  --datadir=/path/to/reth-datadir \
  --proofs-history.storage-path=/path/to/proof-db \
  --proofs-history.window=600000 \
  --proofs-history.prune-batch-size=10000
```

`op-reth proofs unwind`
Manually unwinds the proof history to a specific block. Useful for recovering from corrupted states.

```bash
op-reth proofs unwind \
  --datadir=/path/to/reth-datadir \
  --proofs-history.storage-path=/path/to/proofs-db \
  --target=90
```

### 4. Metrics
A comprehensive Grafana dashboard is available at `etc/grafana/dashboards/op-proof-history.json` to monitor:
-   Syncing speed
-   Sidecar storage size.
-   Pruning performance.
-   Proof generation latency.

Sample metric snapshot available at: https://snapshots.raintank.io/dashboard/snapshot/bzYXscOCugsxO6C2bzFB1XbskxG0KFdo

## Performance

We benchmarked the sidecar on Base Sepolia to validate latency and throughput under load.

Metric | Result
-- | --
Avg Latency | 	15 ms
Throughput	|   ~5,000 req/sec

Benchmark Configuration
- Network: Base Sepolia (Local Node)
- Target: WETH Contract (0x420...0006)
- Range: ~700k blocks (34,011,476 to 34,704,213)
- Load: 10 concurrent workers, 100 requests per block iteration.

The test script iterates through the block range, spawning 10 concurrent workers. Each worker selects an address round-robin from a pre-defined set, dynamically calculates the storage slot for balanceOf[address], and sends an eth_getProof request.

Visual Proof:
- [Grafana Snapshot: Proof Metrics](https://snapshots.raintank.io/dashboard/snapshot/l74zCP4SXr1qcOR2RWFEiscZnDxGla8Z)
- [Grafana Snapshot: Reth Metrics](https://snapshots.raintank.io/dashboard/snapshot/DRoQMVF0m13d4tMRjhoAzHdfbjBA0eql)

## Limitations

- **High Storage Footprint**: The versioned state model trades storage space for instant computation. Storing versioned Merkle Trie nodes (hashes and branch paths) for every block modification is significantly more storage-intensive than the flat state diffs used by the main node.
- **Forward-Only Availability**: The sidecar implements a "record-forward" strategy. It cannot generate proofs for blocks prior to the sidecar's initialization; it does not backfill history.
- **Pruning & IOPS**: Pruning old history is a random-write intensive operation. High-performance NVMe storage is required to ensure the pruner can keep pace with the chain's growth on high-throughput networks.