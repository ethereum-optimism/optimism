# Derivation Iterator

## Objective

Derive L2 payload attributes from L1 data one block at a time, equivalent in
behavior to the existing streaming pipeline in `op-node/rollup/derive`, but
without I/O, caching, or state access beyond what the caller provides.

```go
d, _ := NewDeriver(cfg, l1ChainConfig, lgr, safeHead, sysConfig)
d.AddL1Block(l1Blocks...)
attrs, l1Ref, err := d.Next(safeHead)
```

The `Deriver` iterator accepts L1 blocks incrementally and produces one
`PayloadAttributes` at a time. The caller executes each block on the engine,
then calls `Next` again with the updated safe head.

## Motivation

The existing derivation pipeline is streaming and pull-based: it requests L1
data on demand, maintains internal state across steps, and interleaves I/O with
computation. This makes it difficult to test, reason about, and use in contexts
where all data is already available (ZK provers, auditing tools, replay
utilities).

An earlier batch-mode `PureDerive` function took all L1 data upfront and
returned all derived blocks at once. This didn't match how derivation works in
practice: derive one block, execute on engine, verify, then derive the next. It
also couldn't validate parent hashes (needs L2 block hashes from execution) and
had no mechanism for L1 reorgs.

The iterator solves both: incremental L1 ingestion, one-at-a-time derivation
with full `CheckBatch` validation including parent hash checks, and explicit
reorg handling via `Reset`.

## Scope

**In scope:** Post-Karst derivation only. Karst implies Holocene, Granite,
Fjord, and all prior forks. This simplifies the implementation:

- Single-channel assembly (Holocene rule: one active channel at a time)
- Strict frame ordering (Holocene)
- No span batch overlap handling (Karst rejects overlapping span batches as
  `BatchPast`)

**Out of scope:**
- Pre-Karst derivation
- L2 execution (we produce attributes, not executed blocks)

## API

```go
var ErrNeedL1Data = errors.New("need more L1 data")
var ErrReorg      = errors.New("L1 reorg detected")

func NewDeriver(cfg, l1ChainConfig, lgr, safeHead, sysConfig) (*Deriver, error)

// AddL1Block appends L1 blocks. Must be contiguous with previously added
// blocks. Returns ErrReorg on parent hash mismatch.
func (d *Deriver) AddL1Block(blocks ...L1Input) error

// Next returns the next derived payload attributes and the L1 block they
// were derived from. Returns ErrNeedL1Data when more L1 blocks are needed.
func (d *Deriver) Next(safeHead eth.L2BlockRef) (*eth.PayloadAttributes, eth.L1BlockRef, error)

// Reset clears all state back to the given safe head + system config.
// Used after L1 reorgs. The caller must re-add L1 blocks from the new chain.
func (d *Deriver) Reset(safeHead eth.L2BlockRef, sysConfig eth.SystemConfig)
```

## Architecture

```
               AddL1Block
                   │
                   ▼
L1Input[] ──► frame parsing ──► channel assembly ──► batch decoding ──► CheckBatch ──► attribute building ──► PayloadAttributes
                                       │                                    │
                                 timeout check                      parent hash check
                                 (per L1 block)                     (via safe head)
                                                                          │
                                                              empty batch fallback
                                                            (seq window expired)
```

### Components

| File | Responsibility |
|------|---------------|
| `deriver.go` | `Deriver` iterator: `NewDeriver`, `AddL1Block`, `Next`, `Reset` |
| `channels.go` | Push-based Holocene single-channel assembler |
| `batches.go` | `decodeBatches` (channel → singular batches via upstream decode) |
| `empty_batch.go` | `makeEmptyBatch` (pure function for seq window expiry) |
| `attributes.go` | `buildAttributes` (batch + L1 data → PayloadAttributes) |
| `types.go` | `L1Input`, `l2Cursor`, sentinel errors |

### Next() Flow

1. Try consuming from `pendingBatches`:
   - `CheckBatch` → `BatchAccept`: build attributes, advance cursor, return
   - `CheckBatch` → `BatchPast`: skip, try next batch
   - `CheckBatch` → `BatchDrop`: discard remaining channel batches
   - `CheckBatch` → `BatchUndecided`: return `ErrNeedL1Data`
2. Process more L1 blocks (`l1Pos < len(l1Blocks)`):
   - Process config logs, check channel timeout
   - Parse frames → assemble channel → if ready, decode into `pendingBatches`
   - If got pending batches, go to step 1
   - After each L1 block, check for empty batches (seq window expired)
3. Return `ErrNeedL1Data`

### Empty Batch Generation

When no batcher data covers a time range and the sequencing window expires
(`currentL1.Number > cursor.L1Origin.Number + SeqWindowSize`), the pipeline
generates one empty batch to maintain L2 liveness. Epoch advancement follows
the rule: advance to the next L1 origin when the L2 timestamp >= the next L1
block's timestamp.

## Batch Validation

Batch validation is delegated entirely to upstream `derive.CheckBatch`, which
dispatches to `checkSingularBatch`. Since `Next` receives a full
`eth.L2BlockRef` with `Hash`, `checkSingularBatch` validates
`batch.ParentHash != l2SafeHead.Hash` — solving the parent hash problem that
the earlier batch-mode approach had to defer.

`CheckBatch` expects `l1Blocks[0]` to match `safeHead.L1Origin`. The deriver
computes the starting index dynamically:

```go
startIdx := safeHead.L1Origin.Number - d.firstL1Num
l1BlocksForCheck := d.l1Origins[startIdx:]
```

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

The implementation reuses these upstream types and functions (aliased as
`opderive` to avoid naming conflict with this package):
- `opderive.ParseFrames`, `opderive.Channel`, `opderive.Frame`
- `opderive.BatchReader`, `opderive.GetSingularBatch`, `opderive.DeriveSpanBatch`
- `opderive.CheckBatch`, `opderive.CheckSpanBatchPrefix`
- `opderive.L1InfoDeposit`
- `opderive.ProcessSystemConfigUpdateLogEvent`
- `rollup.Config`, `rollup.ChainSpec`
- `eth.PayloadAttributes`, `eth.L1BlockRef`, `eth.L2BlockRef`, `eth.SystemConfig`

## Testing

Unit tests cover each component in isolation:
- `channels_test.go`: Channel assembly, timeout, frame ordering
- `attributes_test.go`: Payload attribute construction
- `types_test.go`: Cursor advancement, empty batch detection
- `batches_test.go`: Batch decoding from channel data
- `empty_batch_test.go`: Empty batch generation (same epoch, epoch advance, missing L1)
- `deriver_test.go`: Iterator integration tests (single batch, incremental L1,
  empty batches, reorg detection, reorg reset, channel timeout, invalid batch
  drop, parent hash check, pre-Karst rejection, multi-channel multi-epoch)
