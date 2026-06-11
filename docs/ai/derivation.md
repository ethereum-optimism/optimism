# Derivation Pipeline Development

This document provides guidance for AI agents working with the derivation pipeline
in the Optimism monorepo — the logic that reconstructs L2 state from L1 data.
See [go-dev.md](go-dev.md) for general Go build, test, and lint workflow and
[rust-dev.md](rust-dev.md) for the Rust workflow.

The same derivation logic is implemented twice:

- **Go (op-node)**: the reference consensus-layer node, under `op-node/rollup/`.
- **Rust (kona)**: **kona-node** (`rust/kona/bin/node`) is the Rust consensus-layer node;
  **kona-client** (`rust/kona/bin/client`) is the Rust fault proof program. Both drive the
  same derivation pipeline, implemented in the **kona-derive** crate
  (`rust/kona/crates/protocol/derive`).

## Scope

The derivation pipeline lives primarily under `op-node/rollup/` (Go) and
`rust/kona/crates/protocol/derive` (Rust). It reads deposits, batches, and channel data from
L1 and applies them to produce the L2 chain.

## Key Concepts

The pipeline is a series of stages, each pulling from the one below it. From L1 data to L2
blocks:

1. **L1 block traversal**: for each L1 block, read the deposit transaction logs and the
   `SystemConfig` update logs (from the receipts), and collect the batcher transactions
   addressed to the batch inbox — their payload is either calldata or blobs depending on the
   chain's data-availability mode.
2. **Frame extraction**: parse the batcher transactions into frames.
3. **Channel assembly**: assemble frames into channels (a channel may span multiple frames and
   multiple L1 blocks).
4. **Channel decompression**: decompress each complete channel into a stream of batches.
5. **Payload-attributes derivation**: derive L2 payload attributes from the batches, prepending
   the deposit transactions and the L1-info/system-config deposit for the block.
6. **Consolidation vs. execution**: if the existing unsafe chain already matches the derived
   payload attributes (**consolidation**), the block is promoted to safe and derivation
   progresses to the next batch / L1 block without re-executing. Otherwise the payload
   attributes are sent to the execution layer to build and execute the block. In Go this
   matching lives in `op-node/rollup/attributes/` (`AttributesMatchBlock`).

Other recurring concerns:

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
