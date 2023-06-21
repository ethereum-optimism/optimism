# Span-batches

<!-- All glossary references in this file. -->
[g-deposit-tx-type]: glossary.md#deposited-transaction-type

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Introduction](#introduction)
- [Span batch format](#span-batch-format)
- [Integration](#integration)
  - [Channel Reader (Batch Decoding)](#channel-reader-batch-decoding)
  - [Batch Queue](#batch-queue)
  - [Batcher](#batcher)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

> The span-batches spec is experimental :shipit:
>
> *this feature is in active R&D and not yet part of any hard fork

## Introduction

Span-batches reduce overhead of OP-stack chains.
This enables sparse and low-throughput OP-stack chains.

The overhead is reduced by representing a span of
consecutive L2 blocks in a more efficient manner,
while preserving the same consistency checks as regular batch data.

Note that the [channel](./derivation.md#channel-format) and
[frame](./derivation.md#frame-format) formats stay the same:
data slicing, packing and multi-transaction transport is already optimized.

The overhead in the [V0 batch format](./derivation.md) comes from:

- The meta-data attributes are repeated for every L2 block, while these are mostly implied already:
  - parent hash (32 bytes)
  - L1 epoch: blockhash (32 bytes) and block number (~4 bytes)
  - timestamp (~4 bytes)
- The organization of block data is inefficient:
  - Similar attributes are far apart, diminishing any chances of effective compression.
  - Random data like hashes are positioned in-between the more compressible application data.
- The RLP encoding of the data adds unnecessary overhead
  - The outer list does not have to be length encoded, the attributes are known
  - Fixed-length attributes do not need any encoding
  - The batch-format is static and can be optimized further
- Remaining meta-data for consistency checks can be optimized further:
  - The metadata only needs to be secure for consistency checks. E.g. 20 bytes of a hash may be enough.

Span-batches address these inefficiencies, with a new batch format version.

## Span batch format

Note that span-batches, unlike previous V0 batches,
encode *a range of consecutive* L2 blocks at the same time.

Introduce version `1` to the [batch-format](./derivation.md#batch-format) table:

| `batch_version` | `content`           |
|-----------------|---------------------|
| 1               | `prefix ++ payload` |

Notation:
`++`: concatenation of byte-strings.
`anchor`: first L2 block in the span
`uvarint`: unsigned Base128 varint, as defined in [protobuf spec]

[protobuf spec]: https://protobuf.dev/programming-guides/encoding/#varints

Where:

- `prefix = rel_timestamp ++ parent_check ++ l1_origin_check`
  - `rel_timestamp`: relative time since genesis, i.e. `anchor.timestamp - config.genesis.timestamp`.
  - `parent_check`: first 20 bytes of parent hash, i.e. `anchor.parent_hash[:20]`.
  - `l1_origin_check`: to ensure the intended L1 origins of this span of
        L2 blocks are consistent with the L1 chain, the blockhash of the last L1 origin is referenced.
        The hash is truncated to 20 bytes for efficiency, i.e. `anchor.l1_origin.hash[:20]`.
- `payload = block_count ++ block_tx_counts ++ tx_data_headers ++ tx_data ++ tx_sigs`:
  - `block_count`: `uvarint` number of L2 blocks.
  - `origin_bits`: bitlist of `block_count` bits, right-padded to a multiple of 8 bits:
    1 bit per L2 block, indicating if the L1 origin changed this L2 block.
  - `block_tx_counts`: for each block, a `uvarint` of `len(block.transactions)`.
  - `tx_data_headers`: lengths of each `tx_data` entry, encodes as concatenated `uvarint` entries, (empty if there are
    no entries).
  - `tx_data`: [EIP-2718] encoded transactions.
    - The `tx_signature` is truncated from each [EIP-2718] encoded tx. To be reconstructed from `tx_sigs`.
      - `legacy`: starting at `v` RLP field
      - `1` ([EIP-2930]): starting at `signatureYParity` RLP field
      - `2` ([EIP-1559]): starting at `signature_y_parity` RLP field
  - `tx_sigs`: concatenated list of transaction signatures:
    - `v`, or `y_parity`, is encoded as `uvarint` (some legacy transactions combine the chain ID)
    - `r` is encoded as big-endian `uint256`
    - `s` is encoded as big-endian `uint256`

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

[EIP-2930]: https://eips.ethereum.org/EIPS/eip-2930

[EIP-1559]: https://eips.ethereum.org/EIPS/eip-1559

> **TODO research/experimentation questions:**
>
> - `tx_data` entries may be split up completely and tx attributes could be grouped into individual arrays, similar to
    signatures.
    > This may add more complexity, but organize data for improved compression.
> - Backtesting: using this format, how are different types of chain history affected? Improved or not? And by what
    margin?

## Integration

### Channel Reader (Batch Decoding)

The Channel Reader decodes the span-batch, as described in the [span-batch format](#span-batch-format).

A set of derived attributes is computed, cached with the decoded result:

- `l2_blocks_count`: number of L2 blocks in the span-batch
- `start_timestamp`: `config.genesis.timestamp + batch.rel_timestamp`
- `epoch_end`:

### Batch Queue

A span-batch is buffered as a singular large batch,
by its starting timestamp (transformed `rel_timestamp`).

Span-batches share the same queue with v0 batches: batches are processed in L1 inclusion order.

A set of modified validation rules apply to the span-batches.

Rules are enforced with the [contextual definitions](./derivation.md#batch-queue) as v0-batch validation:
`batch`, `epoch`, `inclusion_block_number`, `next_timestamp`, `next_epoch`, `batch_origin`

Span-batch rules, in validation order:

- `batch.start_timestamp > next_timestamp` -> `future`: i.e. the batch must be ready to process.
- `batch.start_timestamp < next_timestamp` -> `drop`: i.e. the batch must not be too old.
- `batch.parent_check != safe_l2_head.hash[:20]` -> `drop`: i.e. the checked part of the parent hash must be equal
  to the L2 safe head block hash.
- Sequencing-window checks:
  - Note: The sequencing window is enforced for the *batch as a whole*:
    if the batch was partially invalid instead, it would drop the oldest L2 blocks,
    which makes the later L2 blocks invalid.
  - Variables:
    - `origin_changed_bit = origin_bits[0]`: `true` if the first L2 block changed its L1 origin, `false` otherwise.
    - `start_epoch_num = safe_l2_head.origin.block_number + (origin_changed_bit ? 1 : 0)`
    - `end_epoch_num = safe_l2_head.origin.block_number + sum(origin_bits)`: block number of last referenced L1 origin
  - Rules:
    - `start_epoch_num + sequence_window_size < inclusion_block_number` -> `drop`:
      i.e. the batch must be included timely.
    - `end_epoch_num < epoch.number` -> `future`: i.e. all referenced L1 epochs must be there.
    - `end_epoch_num == epoch.number`:
      - If `batch.l1_origin_check != epoch.hash[:20]` -> `drop`: verify the batch is intended for this L1 chain.
    - `end_epoch_num > epoch.number` -> `drop`: must have been duplicate batch,
      we may be past this L1 block in the safe L2 chain.
- Max Sequencer time-drift checks:
  - Note: The max time-drift is enforced for the *batch as a whole*, to keep the possible output variants small.
  - Variables:
    - `block_input`: an L2 block from the span-batch,
      with L1 origin as derived from the `origin_bits` and now established canonical L1 chain.
    - `next_epoch` is relative to the `block_input`,
      and may reach to the next origin outside of the L1 origins of the span.
  - Rules:
    - For each `block_input` that can be read from the span-batch:
      - `block_input.timestamp < block_input.origin.time` -> `drop`: enforce the min L2 timestamp rule.
      - `block_input.timestamp > block_input.origin.time + max_sequencer_drift`: enforce the L2 timestamp drift rule,
        but with exceptions to preserve above min L2 timestamp invariant:
        - `len(block_input.transactions) == 0`:
          - `epoch.number == batch.epoch_num`:
            this implies the batch does not already advance the L1 origin,
            and must thus be checked against `next_epoch`.
            - If `next_epoch` is not known -> `undecided`:
              without the next L1 origin we cannot yet determine if time invariant could have been kept.
            - If `batch.timestamp >= next_epoch.time` -> `drop`:
              the batch could have adopted the next L1 origin without breaking the `L2 time >= L1 time` invariant.
        - `len(batch.transactions) > 0`: -> `drop`:
          when exceeding the sequencer time drift, never allow the sequencer to include transactions.
- And for all transactions:
  - `drop` if the `batch.transactions` list contains a transaction
    that is invalid or derived by other means exclusively:
    - any transaction that is empty (zero length `tx_data`)
    - any [deposited transactions][g-deposit-tx-type] (identified by the transaction type prefix byte in `tx_data`)

Once validated, the batch-queue then emits a block-input for each of the blocks included in the span-batch.
The next derivation stage is thus only aware of individual block inputs, similar to the previous V0 batch,
although not strictly a "v0 batch" anymore.

### Batcher

Instead of transforming L2 blocks into batches,
the blocks should be buffered to form a span-batch.

Ideally the L2 blocks are buffered as block-inputs, to maximize the span of blocks covered by the span-batch:
span-batches of single L2 blocks do not increase efficiency as much as with larger spans.

This means that the `(c *channelBuilder) AddBlock` function is changed to
not directly call `(co *ChannelOut) AddBatch` but defer that until a minimum number of blocks have been buffered.

Output-size estimation of the queued up blocks is not possible until the span-batch is written to the channel.
Past a given number of blocks, the channel may be written for estimation, and then re-written if more blocks arrive.

The [batcher functionality](./batcher.md) stays the same otherwise: unsafe blocks are transformed into batches,
encoded in compressed channels, and then split into frames for submission to L1.
Batcher implementations can implement different heuristics and re-attempts to build the most gas-efficient data-txs.
