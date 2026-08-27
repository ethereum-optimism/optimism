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

## Rollup config

Both clients' derivation rules are configured by consensus parameters in `rollup.Config` (Go) /
`RollupConfig` (Rust), which can be loaded from the
[superchain-registry](https://github.com/ethereum-optimism/superchain-registry). Adding a
field that comes from the registry means wiring it through **every** ingestion enumeration on
**both** clients — not just the config struct and the op-deployer/deploy-config
(`DeployConfig.RollupConfig`) path:

- **op-node (Go)**: the TOML-decoded `superchain.HardforkConfig` (`op-core/superchain`) **and**
  the `superchain.ChainConfig` → `rollup.Config` conversion in `op-node/rollup/superchain.go`
  (`applyHardforks` / `rollupConfigFromRegistry`).
- **kona (Rust)**: `HardForkConfig` / `ChainConfig::as_rollup_config`
  (`rust/kona/crates/protocol/genesis`).

Two guards fail loudly on a field left unwired:

- **Strict decoding** rejects registry keys that no struct field models —
  `jsonutil.DecodeTOMLStrict` (Go, used by `op-core/superchain`) and
  `#[serde(deny_unknown_fields)]` (kona's `ChainConfig` / `HardForkConfig`). A registry bump that
  adds an unmodeled field fails to load until the struct consumes it. kona's `RollupConfig` and the
  nested types it holds are strict the same way, so an operator-supplied `rollup.json` carrying a
  key kona does not model fails to load instead of being silently dropped. op-node's
  `ParseRollupConfig` is lenient, which makes the asymmetry cut both ways: a **new** `rollup.Config`
  field has to be modeled in kona before a config carrying it loads in kona-node
  (`TestKonaRollupConfigFixture`, `op-node/rollup/kona_rollup_config_test.go`, fails when the two
  drift apart), and a **retired** one that op-node simply stops modeling has to stay modeled-and-
  ignored in kona, or configs still carrying it stop loading there while op-node keeps accepting
  them.
- **A reflection completeness test** (`TestRollupConfigFromRegistry_AllFieldsSet`) asserts every
  `rollup.Config` field is populated from a fully-populated `ChainConfig`, catching a field that
  is modeled but never copied in the conversion.

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

### Validate transactions after span decomposition

For post-Holocene derivation, transaction-list validation belongs to the batch stage, after a span
batch has been decomposed and its singular batches are streamed one at a time. In particular,
activation-gated transaction rules must use the streamed singular batch's timestamp. Do not inspect
transactions while decoding or constructing a `SpanBatch`, and keep post-Holocene whole-span
processing limited to prefix and extraction checks. A span may cross a fork boundary; only singular
batches emitted after the safe head are candidates for validation.

Add new per-block transaction rules to `checkSingularBatch` (op-node) and
`SingleBatch::check_batch` (kona), where singular batches from both wire formats converge. Rejecting
during `DeriveSpanBatch` can discard valid later elements before the batch stage has selected the
singular batches that actually apply. The legacy pre-Holocene batch queue has no singular-streaming
batch stage, so it still performs its historical full-span transaction checks. Keep those legacy
checks aligned between op-node and Kona.

## Cross-client wire-format parity

Both clients decode the same batcher-controlled bytes, so their decoders must accept **exactly**
the same byte set. A byte string that one client decodes and the other rejects splits derivation on
identical L1 data.

- **The spec is the reference; op-node is the incumbent.** Decide what is correct from the
  [specs](https://specs.optimism.io). op-node additionally defines what OP Mainnet currently does,
  so where the spec is silent or ambiguous its behavior is the tie-breaker — but a decoder that
  contradicts the spec, or looks outright buggy, is a finding to raise rather than something to
  mirror into kona. Either way, take a spec/op-node disagreement to the user before encoding a
  choice in either client.
- **Verify a codec's accept-set, don't infer it from the format name.** Wire formats come in
  families that differ on exactly the inputs a batcher controls. Span-batch `uvarint` is a protobuf
  Base128 varint, whose non-minimal encodings are valid; the `unsigned-varint` crate implements the
  minimal-only multiformats variant instead and rejects them. Span-batch fields go through
  `read_uvarint` (`rust/kona/crates/protocol/protocol/src/batch/varint.rs`), a port of Go's
  `binary.ReadUvarint`. Before trusting any decoder on this path, diff its accept-set against
  op-node's over a generated corpus — a spec citation does not distinguish two families.
- **Pin every decision in both suites.** When changing either decoder, add the same byte vector to
  the kona and op-node tests so the pair stays locked together.

## Testing Requirements

- Unit tests for every pipeline stage.
- Reorg simulation tests for any change to reorg handling.
- End-to-end derivation tests with synthetic L1 data.
- Benchmark tests for batch-processing throughput.
