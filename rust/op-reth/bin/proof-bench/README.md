# OP Reth Proof Benchmark Tool

`op-reth-proof-bench` benchmarks the performance of the `eth_getProof` RPC method on OP Stack nodes. It iterates through a range of blocks, sending concurrent proof requests for the canonical WETH predeploy (`0x4200000000000000000000000000000000000006`), and reports latency and throughput metrics.

The workload is OP Stack-specific: it queries `balanceOf` storage slots at Solidity mapping position `3` for a fixed set of addresses. It is not a configurable benchmark for arbitrary Ethereum contracts or storage layouts.

## Features

- **Concurrent Execution:** Sends multiple requests in parallel to stress test the RPC.
- **Detailed Reporting:**
  - Real-time per-block stats (Req/s, P95 Latency, Min/Max).
  - Final summary with histogram-based percentiles (P50, P95, P99).
- **Configurable Load:** Configure worker count, request count per block, and block step.
- **Robustness:** Handles network errors gracefully and reports error counts.

## Installation

This tool is part of the `op-reth` workspace. You can run it directly using Cargo.

```bash
# Build and run directly
cargo run -p op-reth-proof-bench -- --help
```

## Usage

### Basic Example

Benchmark 100 consecutive blocks, from block `10,000,000` through block `10,000,099`, against a local node:

```bash
cargo run --release -p op-reth-proof-bench -- \
  --rpc http://localhost:8545 \
  --from 10000000 \
  --to 10000099 \
  --step 1
```

Both range endpoints are inclusive.

### Advanced Usage

Stress test a remote node with higher concurrency:

```bash
cargo run --release -p op-reth-proof-bench -- \
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
| `--step` | `10000` | Increment between queried block numbers. |
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
