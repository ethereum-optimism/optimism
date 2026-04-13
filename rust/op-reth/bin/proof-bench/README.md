# Reth Proof Benchmark Tool

`op-reth-proof-bench` is a specialized CLI tool designed to benchmark the performance of the `eth_getProof` RPC method on Optimism/Ethereum nodes. It iterates through a range of blocks, sending concurrent proof requests to valid addresses, and reports detailed latency and throughput metrics.

## Features

- **Concurrent Execution:** Sends multiple requests in parallel to stress test the RPC.
- **Detailed Reporting:**
  - Real-time per-block stats (Req/s, P95 Latency, Min/Max).
  - Final summary with histogram-based percentiles (P50, P95, P99).
- **Customizable Workload:** Configure worker count, request count per block, and block step.
- **Robustness:** Handles network errors gracefully and reports error counts.

## Installation

This tool is part of the `op-reth` workspace. You can run it directly using Cargo.

```bash
# Build and run directly
cargo run -p reth-proof-bench -- --help
```

## Usage

### Basic Example

Benchmark 100 blocks from block `10,000,000` to `10,000,100` against a local node:

```bash
cargo run --release -p reth-proof-bench -- \
  --rpc http://localhost:8545 \
  --from 10000000 \
  --to 10000100
```

### Advanced Usage

Stress test a remote node with higher concurrency:

```bash
cargo run --release -p reth-proof-bench -- \
  --rpc http://remote-node:8545 \
  --from 4000000 \
  --to 4100000 \
  --step 10000 \
  --reqs 50 \
  --workers 10
```

### Arguments

| Flag | Default | Description |
|------|---------|-------------|
| `--rpc` | `http://localhost:8545` | The HTTP RPC endpoint of the node. |
| `--from` | **Required** | Start block number. |
| `--to` | **Required** | End block number. |
| `--step` | `10000` | Number of blocks to skip between benchmark iterations. |
| `--reqs` | `10` | Number of `eth_getProof` requests to send *per block*. |
| `--workers` | `2` | Number of concurrent async workers to run. |

## Output Example

```text
Block      | Req/s      | Min(ms)    | P95(ms)    | Max(ms)    | Errors    
---------------------------------------------------------------------------
36441154   | 245.50     | 25.12      | 45.20      | 55.10      | 0         
36451154   | 230.10     | 26.05      | 48.10      | 60.15      | 0         

---------------------------------------------------------------------------
Summary:
Total Requests:      100
Total Time:          0.85s
Throughput (Req/s):  117.65
Total Errors:        0
-----------------------------------
Min Latency:         25.12 ms
Median Latency:      32.00 ms
P95 Latency:         48.10 ms
P99 Latency:         60.15 ms
Max Latency:         60.15 ms
---------------------------------------------------------------------------
```
