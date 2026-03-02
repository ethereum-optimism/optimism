# Pure Derivation Pipeline

## Objective

Implement a pure function that derives L2 payload attributes from L1 data,
equivalent in behavior to the existing streaming pipeline in
`op-node/rollup/derive`, but without I/O, caching, or state access.

```
PureDerive(cfg, l1ChainConfig, logger, safeHead, sysConfig, l1Blocks) → []DerivedBlock
```

Given the same inputs, the function always produces the same outputs. The caller
provides all L1 data upfront; the function never fetches anything.

## Motivation

The existing derivation pipeline is streaming and pull-based: it requests L1
data on demand, maintains internal state across steps, and interleaves I/O with
computation. This makes it difficult to test, reason about, and use in contexts
where all data is already available (ZK provers, auditing tools, replay
utilities).

A pure function is deterministic, composable, and trivially testable.

## Scope

**In scope:** Post-Karst derivation only. Karst implies Holocene, Granite,
Fjord, and all prior forks. This simplifies the implementation:

- No `BatchFuture` or `BatchUndecided` (Holocene semantics: future → drop,
  undecided conditions don't arise with complete L1 data)
- No span batch overlap handling (Karst rejects overlapping span batches as
  `BatchPast`)
- Single-channel assembly (Holocene rule: one active channel at a time)
- Strict frame ordering (Holocene)

**Out of scope:**
- Pre-Karst derivation
- Pipeline reset / reorg detection (caller responsibility)
- L2 execution (we produce attributes, not executed blocks)

## Architecture

```
L1Input[] ──► frame parsing ──► channel assembly ──► batch decoding ──► batch validation ──► attribute building ──► DerivedBlock[]
                                       │
                                 timeout check
                                 (per L1 block)
```

### Components

| File | Responsibility |
|------|---------------|
| `derive.go` | `PureDerive` entry point, main loop over L1 blocks |
| `channels.go` | Push-based Holocene single-channel assembler |
| `batches.go` | `decodeBatches` (channel → singular batches), `validateBatch` |
| `attributes.go` | `buildAttributes` (batch + L1 data → PayloadAttributes) |
| `types.go` | `L1Input`, `DerivedBlock`, `l2Cursor` |

### Main Loop (derive.go)

For each L1 block:
1. Process system config update logs
2. Check channel timeout (fork-aware via `spec.ChannelTimeout`)
3. Parse frames from batcher transactions
4. Assemble frames into channels
5. When a channel completes: decode batches, validate each, build attributes
6. After processing all channels: generate empty batches if the sequencing
   window has expired

### Empty Batch Generation

When no batcher data covers a time range and the sequencing window expires
(`currentL1.Number > cursor.L1Origin.Number + SeqWindowSize`), the pipeline
generates empty batches to maintain L2 liveness. Epoch advancement follows the
rule: advance to the next L1 origin when the L2 timestamp >= the next L1
block's timestamp.

## Behavioral Equivalence

The implementation must match `checkSingularBatch` in
`op-node/rollup/derive/batches.go` for all checks that don't require L2 state.

### Upstream Check Mapping

| # | Upstream Check | Pure Implementation | Notes |
|---|---------------|-------------------|-------|
| 1 | `len(l1Blocks) == 0` → `BatchUndecided` | N/A | We always have all L1 data |
| 2 | `timestamp > next` → `BatchFuture`/`BatchDrop` | `BatchDrop` | Holocene always active (implied by Karst) |
| 3 | `timestamp < next` → `BatchDrop`/`BatchPast` | `BatchPast` | Holocene always active |
| 4 | Parent hash mismatch → `BatchDrop` | Deferred | Stored in `DerivedBlock.ExpectedParentHash` for post-execution verification |
| 5 | Sequence window expired → `BatchDrop` | `epochNum + SeqWindowSize < l1InclusionNum` → `BatchDrop` | Equivalent |
| 6a | Epoch too old → `BatchDrop` | `epochNum < cursor.L1Origin.Number` → `BatchDrop` | Equivalent |
| 6b | Epoch is next but no L1 data → `BatchUndecided` | N/A | We always have all L1 data |
| 6c | Epoch too far ahead → `BatchDrop` | `epochNum > cursor.L1Origin.Number+1` → `BatchDrop` | Equivalent |
| 7 | Epoch hash mismatch → `BatchDrop` | Look up origin, compare hash → `BatchDrop` | Equivalent |
| 8 | Timestamp < L1 origin time → `BatchDrop` | `batch.Timestamp < batchOrigin.Time` → `BatchDrop` | Equivalent |
| 9 | Fork activation block with txs → `BatchDrop` | Jovian, Karst, Interop checks → `BatchDrop` | Equivalent |
| 10 | Sequencer drift exceeded → `BatchDrop` | Same logic with empty batch exception → `BatchDrop` | Equivalent |
| 11a | Empty transaction → `BatchDrop` | `len(txBytes) == 0` → `BatchDrop` | Equivalent |
| 11b | Deposit transaction → `BatchDrop` | `txBytes[0] == DepositTxType` → `BatchDrop` | Equivalent |
| 11c | SetCode before Isthmus → `BatchDrop` | `!isIsthmus && txBytes[0] == SetCodeTxType` → `BatchDrop` | Equivalent |
| 12 | All pass → `BatchAccept` | → `BatchAccept` | Equivalent |

### Intentional Differences

1. **Parent hash validation (check #4):** Deferred to post-execution. The pure
   function has no L2 block hashes. The caller can verify
   `DerivedBlock.ExpectedParentHash` against actual execution results.

2. **No `BatchUndecided` or `BatchFuture`:** With Holocene active and all L1
   data provided, these states cannot occur.

3. **Span batch overlaps:** Under Karst, `CheckSpanBatchPrefix` rejects
   overlapping span batches as `BatchPast` (upstream treats them as errors
   pre-Karst). This is the one behavioral change vs pre-Karst upstream.

4. **`BatchPast` handling:** In the main loop, `BatchPast` batches are skipped
   (`continue`), not flushed. `BatchDrop` and other non-accept results cause a
   `break` that flushes the remaining batches from the current channel. This
   matches Holocene semantics where past batches are harmless leftovers.

### Attribute Building Equivalence

`buildAttributes` matches `derive.AttributesDeposited` for:
- L1 info deposit transaction (via `derive.L1InfoDeposit`)
- User deposits at epoch boundaries
- Sequencer transactions from the batch
- Canyon withdrawals, Ecotone parent beacon root
- Holocene EIP-1559 params, Jovian MinBaseFee
- Gas limit from system config
- `NoTxPool: true`

Not included: network upgrade transactions (NUTs) for pre-Karst forks, since
all pre-Karst forks are already active. Future forks with NUTs must be added.

## Dependencies on Upstream

The implementation reuses these upstream types and functions:
- `derive.ParseFrames`, `derive.Channel`, `derive.Frame`
- `derive.BatchReader`, `derive.GetSingularBatch`, `derive.DeriveSpanBatch`
- `derive.CheckSpanBatchPrefix`
- `derive.L1InfoDeposit`
- `derive.ProcessSystemConfigUpdateLogEvent`
- `rollup.Config`, `rollup.ChainSpec`
- `eth.PayloadAttributes`, `eth.L1BlockRef`, `eth.L2BlockRef`, `eth.SystemConfig`

## Testing

Unit tests cover each component in isolation:
- `batches_test.go`: Batch decoding and all `validateBatch` rejection paths
- `channels_test.go`: Channel assembly, timeout, frame ordering
- `attributes_test.go`: Payload attribute construction
- `types_test.go`: Cursor advancement, empty batch detection
- `derive_test.go`: Integration tests for `PureDerive` (single batch, empty
  epochs, multi-channel, channel timeout, invalid batch skip, pre-Karst
  rejection, L1 range validation)
