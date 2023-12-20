# Span-batches

<!-- All glossary references in this file. -->
[g-deposit-tx-type]: glossary.md#deposited-transaction-type

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Introduction](#introduction)
- [Span batch format](#span-batch-format)
  - [Max span-batch size](#max-span-batch-size)
  - [Future batch-format extension](#future-batch-format-extension)
- [Span batch Activation Rule](#span-batch-activation-rule)
- [Optimization Strategies](#optimization-strategies)
  - [Truncating information and storing only necessary data](#truncating-information-and-storing-only-necessary-data)
  - [`tx_data_headers` removal from initial specs](#tx_data_headers-removal-from-initial-specs)
  - [`Chain ID` removal from initial specs](#chain-id-removal-from-initial-specs)
  - [Reorganization of constant length transaction fields](#reorganization-of-constant-length-transaction-fields)
  - [RLP encoding for only variable length fields](#rlp-encoding-for-only-variable-length-fields)
  - [Store `y_parity` and `protected_bit` instead of `v`](#store-y_parity-and-protected_bit-instead-of-v)
  - [Adjust `txs` Data Layout for Better Compression](#adjust-txs-data-layout-for-better-compression)
  - [`fee_recipients` Encoding Scheme](#fee_recipients-encoding-scheme)
- [How derivation works with Span Batch?](#how-derivation-works-with-span-batch)
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

[span-batch-format]: #span-batch-format

Note that span-batches, unlike previous singular batches,
encode *a range of consecutive* L2 blocks at the same time.

Introduce version `1` to the [batch-format](./derivation.md#batch-format) table:

| `batch_version` | `content`           |
|-----------------|---------------------|
| 1               | `prefix ++ payload` |

Notation:

- `++`: concatenation of byte-strings
- `span_start`: first L2 block in the span
- `span_end`: last L2 block in the span
- `uvarint`: unsigned Base128 varint, as defined in [protobuf spec]
- `rlp_encode`: a function that encodes a batch according to the [RLP format],
  and `[x, y, z]` denotes a list containing items `x`, `y` and `z`

[protobuf spec]: https://protobuf.dev/programming-guides/encoding/#varints

Standard bitlists, in the context of span-batches, are encoded as big-endian integers,
left-padded with zeroes to the next multiple of 8 bits.

Where:

- `prefix = rel_timestamp ++ l1_origin_num ++ parent_check ++ l1_origin_check`
  - `rel_timestamp`: `uvarint` relative timestamp since L2 genesis,
    i.e. `span_start.timestamp - config.genesis.timestamp`.
  - `l1_origin_num`: `uvarint` number of last l1 origin number. i.e. `span_end.l1_origin.number`
  - `parent_check`: first 20 bytes of parent hash, the hash is truncated to 20 bytes for efficiency,
    i.e. `span_start.parent_hash[:20]`.
  - `l1_origin_check`: the block hash of the last L1 origin is referenced.
    The hash is truncated to 20 bytes for efficiency, i.e. `span_end.l1_origin.hash[:20]`.
- `payload = block_count ++ origin_bits ++ block_tx_counts ++ txs`:
  - `block_count`: `uvarint` number of L2 blocks. This is at least 1, empty span batches are invalid.
  - `origin_bits`: standard bitlist of `block_count` bits:
    1 bit per L2 block, indicating if the L1 origin changed this L2 block.
  - `block_tx_counts`: for each block, a `uvarint` of `len(block.transactions)`.
  - `txs`: L2 transactions which is reorganized and encoded as below.
- `txs = contract_creation_bits ++ y_parity_bits ++
        tx_sigs ++ tx_tos ++ tx_datas ++ tx_nonces ++ tx_gases ++ protected_bits`
  - `contract_creation_bits`: standard bitlist of `sum(block_tx_counts)` bits:
    1 bit per L2 transactions, indicating if transaction is a contract creation transaction.
  - `y_parity_bits`: standard bitlist of `sum(block_tx_counts)` bits:
    1 bit per L2 transactions, indicating the y parity value when recovering transaction sender address.
  - `tx_sigs`: concatenated list of transaction signatures
    - `r` is encoded as big-endian `uint256`
    - `s` is encoded as big-endian `uint256`
  - `tx_tos`: concatenated list of `to` field. `to` field in contract creation transaction will be `nil` and ignored.
  - `tx_datas`: concatenated list of variable length rlp encoded data,
    matching the encoding of the fields as in the [EIP-2718] format of the `TransactionType`.
    - `legacy`: `rlp_encode(value, gasPrice, data)`
    - `1`: ([EIP-2930]): `0x01 ++ rlp_encode(value, gasPrice, data, accessList)`
    - `2`: ([EIP-1559]): `0x02 ++ rlp_encode(value, max_priority_fee_per_gas, max_fee_per_gas, data, access_list)`
  - `tx_nonces`: concatenated list of `uvarint` of `nonce` field.
  - `tx_gases`:  concatenated list of `uvarint` of gas limits.
    - `legacy`: `gasLimit`
    - `1`: ([EIP-2930]): `gasLimit`
    - `2`: ([EIP-1559]): `gas_limit`
  - `protected_bits`: standard bitlist of length of number of legacy transactions:
    1 bit per L2 legacy transactions, indicating if transaction is protected([EIP-155]) or not.

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718
[EIP-2930]: https://eips.ethereum.org/EIPS/eip-2930
[EIP-1559]: https://eips.ethereum.org/EIPS/eip-1559
[EIP-155]: https://eips.ethereum.org/EIPS/eip-155

### Max span-batch size

Total size of encoded span batch is limited to `MAX_SPAN_BATCH_SIZE` (currently 10,000,000 bytes,
equal to `MAX_RLP_BYTES_PER_CHANNEL`). Therefore every field size of span batch will be implicitly limited to
`MAX_SPAN_BATCH_SIZE` . There can be at least single span batch per channel, and channel size is limited
to `MAX_RLP_BYTES_PER_CHANNEL` and you may think that there is already an implicit limit. However, having an explicit
limit for span batch is helpful for several reasons. We may save computation costs by avoiding malicious input while
decoding. For example, let's say bad batcher wrote span batch which `block_count = max.Uint64`. We may early return
using the explicit limit, not trying to consume data until EOF is reached. We can also safely preallocate memory for
decoding because we know the upper limit of memory usage.

### Future batch-format extension

This is an experimental extension of the span-batch format, and not activated with the Delta upgrade yet.

Introduce version `2` to the [batch-format](./derivation.md#batch-format) table:

| `batch_version` | `content`           |
|-----------------|---------------------|
| 2               | `prefix ++ payload` |

Where:

- `prefix = rel_timestamp ++ l1_origin_num ++ parent_check ++ l1_origin_check`:
  - Identical to `batch_version` 1
- `payload = block_count ++ origin_bits ++ block_tx_counts ++ txs ++ fee_recipients`:
  - An empty span-batch, i.e. with `block_count == 0`, is invalid and must not be processed.
  - Every field definition identical to `batch_version` 1 except that `fee_recipients` is
    added to support more decentralized sequencing.
  - `fee_recipients = fee_recipients_idxs + fee_recipients_set`
    - `fee_recipients_set`: concatenated list of unique L2 fee recipient address.
    - `fee_recipients_idxs`: for each block,
      `uvarint` number of index to decode fee recipients from `fee_recipients_set`.

## Span batch Activation Rule

The span batch upgrade is activated based on timestamp.

Activation Rule: `upgradeTime != null && span_start.l1_origin.timestamp >= upgradeTime`

`span_start.l1_origin.timestamp` is the L1 origin block timestamp of the first block in the span batch.
This rule ensures that every chain activity regarding this span batch is done after the hard fork.
i.e. Every block in the span is created, submitted to the L1, and derived from the L1 after the hard fork.

## Optimization Strategies

### Truncating information and storing only necessary data

The following fields stores truncated data:

- `rel_timestamp`: We can save two bytes by storing `rel_timestamp` instead of the full `span_start.timestamp`.
- `parent_check` and `l1_origin_check`: We can save twelve bytes by truncating twelve bytes from the full hash,
  while having enough safety.

### `tx_data_headers` removal from initial specs

We do not need to store length per each `tx_datas` elements even if those are variable length,
because the elements itself is RLP encoded, containing their length in RLP prefix.

### `Chain ID` removal from initial specs

Every transaction has chain id. We do not need to include chain id in span batch because L2 already knows its chain id,
and use its own value for processing span batches while derivation.

### Reorganization of constant length transaction fields

`signature`, `nonce`, `gaslimit`, `to` field are constant size, so these were split up completely and
are grouped into individual arrays.
This adds more complexity, but organizes data for improved compression by grouping data with similar data pattern.

### RLP encoding for only variable length fields

Further size optimization can be done by packing variable length fields, such as `access_list`.
However, doing this will introduce much more code complexity, compared to benefiting from size reduction.

Our goal is to find the sweet spot on code complexity - span batch size tradeoff.
I decided that using RLP for all variable length fields will be the best option,
not risking codebase with gnarly custom encoding/decoding implementations.

### Store `y_parity` and `protected_bit` instead of `v`

Only legacy type transactions can be optionally protected. If protected([EIP-155]), `v = 2 * ChainID + 35 + y_parity`.
Else, `v = 27 + y_parity`. For other types of transactions, `v = y_parity`.
We store `y_parity`, which is single bit per L2 transaction.
We store `protected_bit`, which is single bit per L2 legacy type transactions to indicate that tx is protected.

This optimization will benefit more when ratio between number of legacy type transactions over number of transactions
excluding deposit tx is higher.
Deposit transactions are excluded in batches and are never written at L1 so excluded while analyzing.

### Adjust `txs` Data Layout for Better Compression

There are (7 choose 2) * 5! = 2520 permutations of ordering fields of `txs`.
It is not 7! because `contract_creation_bits` must be first decoded in order to decode `tx_tos`.
We experimented to find out the best layout for compression.
It turned out placing random data together(`TxSigs`, `TxTos`, `TxDatas`),
then placing leftovers helped gzip to gain more size reduction.

### `fee_recipients` Encoding Scheme

Let `K` := number of unique fee recipients(cardinality) per span batch. Let `N` := number of L2 blocks.
If we naively encode each fee recipients by concatenating every fee recipients, it will need `20 * N` bytes.
If we manage `fee_recipients_idxs` and `fee_recipients_set`, It will need at most `max uvarint size * N = 8 * N`,
`20 * K` bytes each. If `20 * N > 8 * N + 20 * K` then maintaining an index of fee recipients is reduces the size.

we thought sequencer rotation happens not much often, so assumed that `K` will be much lesser than `N`.
The assumption makes upper inequality to hold. Therefore, we decided to manage `fee_recipients_idxs` and
`fee_recipients_set` separately. This adds complexity but reduces data.

## How derivation works with Span Batch?

- Block Timestamp
  - The first L2 block's block timestamp is `rel_timestamp + L2Genesis.Timestamp`.
  - Then we can derive other blocks timestamp by adding L2 block time for each.
- L1 Origin Number
  - The parent of the first L2 block's L1 origin number is `l1_origin_num - sum(origin_bits)`
  - Then we can derive other blocks' L1 origin number with `origin_bits`
  - `ith block's L1 origin number = (i-1)th block's L1 origin number + (origin_bits[i] ? 1 : 0)`
- L1 Origin Hash
  - We only need the `l1_origin_check`, the truncated L1 origin hash of the last L2 block of Span Batch.
  - If the last block references canonical L1 chain as its origin,
    we can ensure the all other blocks' origins are consistent with the canonical L1 chain.
- Parent hash
  - In V0 Batch spec, we need batch's parent hash to validate if batch's parent is consistent with current L2 safe head.
  - But in the case of Span Batch, because it contains consecutive L2 blocks in the span,
    we do not need to validate all blocks' parent hash except the first block.
- Transactions
  - Deposit transactions can be derived from its L1 origin, identical with V0 batch.
  - User transactions can be derived by following way:
    - Recover `V` value of TX signature from `y_parity_bits` and L2 chainId, as described in optimization strategies.
    - When parsing `tx_tos`, `contract_creation_bits` is used to determine if the TX has `to` value or not.

## Integration

### Channel Reader (Batch Decoding)

The Channel Reader decodes the span-batch, as described in the [span-batch format](#span-batch-format).

A set of derived attributes is computed as described above. Then cached with the decoded result:

### Batch Queue

A span-batch is buffered as a singular large batch,
by its starting timestamp (transformed `rel_timestamp`).

Span-batches share the same queue with v0 batches: batches are processed in L1 inclusion order.

A set of modified validation rules apply to the span-batches.

Rules are enforced with the [contextual definitions](./derivation.md#batch-queue) as v0-batch validation:
`epoch`, `inclusion_block_number`, `next_timestamp`

Definitions:

- `batch` as defined in the [Span batch format section][span-batch-format].
- `prev_l2_block` is the L2 block from the current safe chain,
  whose timestamp is at `span_start.timestamp - l2_block_time`

Span-batch rules, in validation order:

- `batch_origin` is determined like with singular batches:
  - `batch.epoch_num == epoch.number+1`:
    - If `next_epoch` is not known -> `undecided`:
      i.e. a batch that changes the L1 origin cannot be processed until we have the L1 origin data.
    - If known, then define `batch_origin` as `next_epoch`
- `batch_origin.timestamp < span_batch_upgrade_timestamp` -> `drop`:
  i.e. enforce the [span batch upgrade activation rule](#span-batch-activation-rule).
- `span_start.timestamp > next_timestamp` -> `future`: i.e. the batch must be ready to process,
  but does not have to start exactly at the `next_timestamp`, since it can overlap with previously processed blocks,
- `span_end.timestamp < next_timestamp` -> `drop`: i.e. the batch must have at least one new block to process.
- If there's no `prev_l2_block` in the current safe chain -> `drop`: i.e. the timestamp must be aligned.
- `batch.parent_check != prev_l2_block.hash[:20]` -> `drop`:
  i.e. the checked part of the parent hash must be equal to the same part of the corresponding L2 block hash.
- Sequencing-window checks:
  - Note: The sequencing window is enforced for the *batch as a whole*:
    if the batch was partially invalid instead, it would drop the oldest L2 blocks,
    which makes the later L2 blocks invalid.
  - Variables:
    - `origin_changed_bit = origin_bits[0]`: `true` if the first L2 block changed its L1 origin, `false` otherwise.
    - `start_epoch_num = batch.l1_origin_num - sum(origin_bits) + (origin_changed_bit ? 1 : 0)`
    - `end_epoch_num = batch.l1_origin_num`
  - Rules:
    - `start_epoch_num + sequence_window_size < inclusion_block_number` -> `drop`:
      i.e. the batch must be included timely.
    - `start_epoch_num > prev_l2_block.l1_origin.number + 1` -> `drop`:
      i.e. the L1 origin cannot change by more than one L1 block per L2 block.
    - If `batch.l1_origin_check` does not match the canonical L1 chain at `end_epoch_num` -> `drop`:
      verify the batch is intended for this L1 chain.
      - After upper `l1_origin_check` check is passed, we don't need to check if the origin
        is past `inclusion_block_number` because of the following invariant.
      - Invariant: the epoch-num in the batch is always less than the inclusion block number,
        if and only if the L1 epoch hash is correct.
    - `start_epoch_num < prev_l2_block.l1_origin.number` -> `drop`:
      epoch number cannot be older than the origin of parent block
- Max Sequencer time-drift checks:
  - Note: The max time-drift is enforced for the *batch as a whole*, to keep the possible output variants small.
  - Variables:
    - `block_input`: an L2 block from the span-batch,
      with L1 origin as derived from the `origin_bits` and now established canonical L1 chain.
    - `next_epoch`: `block_input.origin`'s next L1 block.
      It may reach to the next origin outside the L1 origins of the span.
  - Rules:
    - For each `block_input` whose timestamp is greater than `safe_head.timestamp`:
      - `block_input.timestamp < block_input.origin.time` -> `drop`: enforce the min L2 timestamp rule.
      - `block_input.timestamp > block_input.origin.time + max_sequencer_drift`: enforce the L2 timestamp drift rule,
        but with exceptions to preserve above min L2 timestamp invariant:
        - `len(block_input.transactions) == 0`:
          - `origin_bits[i] == 0`: `i` is the index of `block_input` in the span batch.
            So this implies the block_input did not advance the L1 origin,
            and must thus be checked against `next_epoch`.
            - If `next_epoch` is not known -> `undecided`:
              without the next L1 origin we cannot yet determine if time invariant could have been kept.
            - If `block_input.timestamp >= next_epoch.time` -> `drop`:
              the batch could have adopted the next L1 origin without breaking the `L2 time >= L1 time` invariant.
        - `len(block_input.transactions) > 0`: -> `drop`:
          when exceeding the sequencer time drift, never allow the sequencer to include transactions.
- And for all transactions:
  - `drop` if the `batch.tx_datas` list contains a transaction
    that is invalid or derived by other means exclusively:
    - any transaction that is empty (zero length `tx_data`)
    - any [deposited transactions][g-deposit-tx-type] (identified by the transaction type prefix byte in `tx_data`)
- Overlapped blocks checks:
  - Note: If the span batch overlaps the current L2 safe chain, we must validate all overlapped blocks.
  - Variables:
    - `block_input`: an L2 block derived from the span-batch.
    - `safe_block`: an L2 block from the current L2 safe chain, at same timestamp as `block_input`
  - Rules:
    - For each `block_input`, whose timestamp is less than `next_timestamp`:
      - `block_input.l1_origin.number != safe_block.l1_origin.number` -> `drop`
      - `block_input.transactions != safe_block.transactions` -> `drop`
        - compare excluding deposit transactions

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
