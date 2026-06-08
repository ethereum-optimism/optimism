# Derivation Pipeline Development

This document provides guidance for AI agents working with the derivation pipeline
in the Optimism monorepo — the op-node logic that reconstructs L2 state from L1 data.
See [go-dev.md](go-dev.md) for general Go build, test, and lint workflow.

## Scope

The derivation pipeline lives primarily under `op-node/rollup/`. It reads deposits,
batches, and channel data from L1 and applies them to produce the L2 chain.

## Key Concepts

- **L1 block traversal**: reading deposits, batches, and channel data from L1.
- **Channel decoding**: reassembling batches from channel frames.
- **Batch processing**: applying batches to produce L2 blocks.
- **Safe head advancement**: updating the safe L2 head as derivation progresses.
- **Reorg handling**: rewinding derivation state on L1 reorgs.

## Invariants

- **Deterministic derivation**: the same L1 data always produces the same L2 chain.
  No randomness, no time-dependent behavior.
- **Safe head bound**: the safe head never advances past finalized L1 data. Safe head
  advancement must verify L1 finality status.
- **Deposit ordering**: all deposits are processed in their L1 inclusion order. Batch
  processing must preserve this ordering.
- **Channel timeout**: channel timeout is enforced to prevent memory exhaustion. Channel
  timeout values must not be modified without protocol review.
- **Reorg unwinding**: reorg handling must correctly unwind all derived state.

## Testing Requirements

- Unit tests for every pipeline stage.
- Reorg simulation tests for any change to reorg handling.
- End-to-end derivation tests with synthetic L1 data.
- Benchmark tests for batch-processing throughput.
