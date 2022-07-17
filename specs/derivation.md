# L2 Chain Derivation Specification

<!-- All glossary references in this file. -->
[g-derivation]: glossary.md#L2-chain-derivation
[g-payload-attr]: glossary.md#payload-attributes
[g-block]: glossary.md#block
[g-exec-engine]: glossary.md#execution-engine
[g-reorg]: glossary.md#re-organization
[g-receipts]: glossary.md#receipt
[g-inception]: glossary.md#L2-chain-inception
[g-deposit-contract]: glossary.md#deposit-contract
[g-deposited]: glossary.md#deposited-transaction
[g-l1-attr-deposit]: glossary.md#l1-attributes-deposited-transaction
[g-user-deposited]: glossary.md#user-deposited-transaction
[g-deposits]: glossary.md#deposits
[g-deposit-contract]: glossary.md#deposit-contract
[g-l1-attr-predeploy]: glossary.md#l1-attributes-predeployed-contract
[g-depositing-call]: glossary.md#depositing-call
[g-depositing-transaction]: glossary.md#depositing-transaction
[g-sequencing]: glossary.md#sequencing
[g-sequencer]: glossary.md#sequencer
[g-sequencing-epoch]: glossary.md#sequencing-epoch
[g-sequencing-window]: glossary.md#sequencing-window
[g-sequencer-batch]: glossary.md#sequencer-batch
[g-l2-genesis]: glossary.md#l2-genesis-block
[g-l2-chain-inception]: glossary.md#L2-chain-inception
[g-batcher-transaction]: glossary.md#batcher-transaction
[g-avail-provider]: glossary.md#data-availability-provider
[g-batcher]: glossary.md#batcher
[g-l2-output]: glossary.md#l2-output
[g-fault-proof]: glosary.md#fault-proof
[g-channel]: glossary.md#channel
[g-channel-frame]: glossary.md#channel-frame
[g-rollup-node]: glossary.md#rollup-node
[g-channel-timeout]: glossary.md#channel-timeout
[g-block-time]: glossary.md#block-time
[g-time-slot]: glossary.md#time-slot
[g-consolidation]: glossary.md#unsafe-block-consolidation
[g-safe-l2-head]: glossary.md#safe-l2-head
[g-unsafe-l2-head]: glossary.md#unsafe-l2-head
[g-unsafe-l2-block]: glossary.md#unsafe-l2-block
[g-unsafe-sync]: glossary.md#unsafe-sync
[g-l1-origin]: glossary.md#l1-origin
[g-deposit-tx-type]: glossary.md#deposited-transaction-type
[g-finalized-l2-head]: glossary.md#finalized-l2-head

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Overview](#overview)
  - [Eager Block Derivation](#eager-block-derivation)
- [Batch Submission](#batch-submission)
  - [Sequencing & Batch Submission Overview](#sequencing--batch-submission-overview)
  - [Batch Submission Wire Format](#batch-submission-wire-format)
    - [Batcher Transaction Format](#batcher-transaction-format)
    - [Frame Format](#frame-format)
    - [Channel Format](#channel-format)
    - [Batch Format](#batch-format)
- [Architecture](#architecture)
  - [L2 Chain Derivation Pipeline](#l2-chain-derivation-pipeline)
    - [L1 Traversal](#l1-traversal)
    - [L1 Retrieval](#l1-retrieval)
    - [Channel Bank](#channel-bank)
    - [Batch Decoding](#batch-decoding)
    - [Batch Buffering](#batch-buffering)
    - [Payload Attributes Derivation](#payload-attributes-derivation)
    - [Engine Queue](#engine-queue)
    - [Resetting the Pipeline](#resetting-the-pipeline)
- [Deriving Payload Attributes](#deriving-payload-attributes)
  - [Deriving the Transaction List](#deriving-the-transaction-list)
  - [Building Individual Payload Attributes](#building-individual-payload-attributes)
- [WARNING: BELOW THIS LINE, THE SPEC HAS NOT BEEN REVIEWED AND MAY CONTAIN MISTAKES](#warning-below-this-line-the-spec-has-not-been-reviewed-and-may-contain-mistakes)
- [Communication with the Execution Engine](#communication-with-the-execution-engine)
- [Handling L1 Re-Orgs](#handling-l1-re-orgs)
  - [Resetting the Engine Queue](#resetting-the-engine-queue)
  - [Resetting Payload Attribute Derivation](#resetting-payload-attribute-derivation)
  - [Resetting Batch Decoding](#resetting-batch-decoding)
  - [Resetting Channel Buffering](#resetting-channel-buffering)
  - [Resetting L1 Retrieval & L1 Traversal](#resetting-l1-retrieval--l1-traversal)
  - [Reorgs Post-Merge](#reorgs-post-merge)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# Overview

> **Note**: the following assumes a single sequencer and batcher. In the future, the design might be adapted to
> accomodate multiple such entities.

[L2 chain derivation][g-derivation] — deriving L2 [blocks][g-block] from L1 data — is one of the main responsability of
the [rollup node][g-rollup-node], both in validator mode, and in sequencer mode (where derivation acts as a sanity check
on sequencing, and enables detecting L1 chain [re-organizations][g-reorg]).

The L2 chain is derived from the L1 chain. In particular, each L1 block is mapped to an L2 [sequencing
epoch][g-sequencing-epoch] comprising multiple L2 blocks. The epoch number is defined to be equal to the corresponding
L1 block number.

To derive the L2 blocks in an epoch `E`, we need the following inputs:

- The L1 [sequencing window][g-sequencing-window] for epoch `E`: the L1 blocks in the range `[E, E + SWS)` where `SWS`
  is the sequencing window size (note that this means that epochs are overlapping). In particular we need:
  - The [batcher transactions][g-batcher-transactions] included in the sequencing window. These allow us to
      reconstruct [sequencer batches][g-sequencer-batch] containing the transactions to include in L2 blocks (each batch
      maps to a single L2 block).
  - The [deposits][g-deposits] made in L1 block `E` (in the form of events emitted by the [deposit
      contract][g-deposit-contract]).
  - The L1 block attributes from L1 block `E` (to derive the [L1 attributes deposited transaction][g-l1-attr-deposit]).
- The state of the L2 chain after the last L2 block of epoch `E - 1`, or — if epoch `E - 1` does not exist — the
  [genesis state][g-l2-genesis] (cf. TODO) of the L2 chain.
  - An epoch `E` does not exist if `E <= L2CI`, where `L2CI` is the [L2 chain inception][g-l2-chain-inception].

> **TODO** specify sequencing window size
> **TODO** specify genesis block / state (in its own document? include/link predeploy.md)

To derive the whole L2 chain from scratch, we simply start with the [L2 genesis state][g-l2-genesis], and the [L2 chain
inception][g-l2-chain-inception] as first epoch, then process all sequencing windows in order. Refer to the
[Architecture section][architecture] for more information on how we implement this in practice.

Each epoch may contain a variable number of L2 blocks (one every `l2_block_time`, 2s on Optimism), at the discretion of
[the sequencer][g-sequencer], but subject to the following constraints for each block:

- `min_l2_timestamp <= block.timestamp < max_l2_timestamp`, where
  - all these values are denominated in seconds
  - `min_l2_timestamp = prev_l2_timestamp + l2_block_time`
    - `prev_l2_timestamp` is the timestamp of the previous L2 block
    - `l2_block_time` is a configurable parameter of the time between L2 blocks (on Optimism, 2s)
  - `max_l2_timestamp = max(l1_timestamp + max_sequencer_drift, min_l2_timestamp + l2_block_time)`
    - `l1_timestamp` is the timestamp of the L1 block associated with the L2 block's epoch
    - `max_sequencer_drift` is the most a sequencer is allowed to get ahead of L1

> **TODO** specify max sequencer drift

Put together, these constraints mean that there must be an L2 block every `l2_block_time` seconds, and that the
timestamp for the first L2 block of an epoch must never fall behind the timestamp of the L1 block matching the epoch.

Post-merge, Ethereum has a fixed [block time][g-block-time] of 12s (though some slots can be skipped). It is thus
expected that, most of the time, each epoch on Optimism will contain `12/2 = 6` L2 blocks. The sequencer can however
lengthen or shorten epochs (subject to above constraints). The rationale is to maintain liveness in case of either a
skipped slot on L1, or a temporary loss of connection to L1 — which requires longer epochs. Shorter epochs are then
required to avoid L2 timestamps drifting further and further ahead of L1.

## Eager Block Derivation

In practice, it is often not necesary to wait for a full sequencing window of L1 blocks in order to start deriving the
L2 blocks in an epoch. Indeed, as long as we are able to reconstruct sequential batches, we can start deriving the
corresponding L2 blocks. We call this *eager block derivation*.

However, in the very worst case, we can only reconstruct the batch for the first L2 block in the epoch by reading the
last L1 block of the sequencing window. This happens when some data for that batch is included in the last L1 block of
the window. In that case, not only can we not derive the first L2 block in the poch, we also can't derive any further L2
block in the epoch until then, as they need the state that results from applying the epoch's first L2 block. (Note that
this only applies to *block* derivation. We can still derive further batches, we just won't be able to create blocks
from them.)

------------------------------------------------------------------------------------------------------------------------

# Batch Submission

## Sequencing & Batch Submission Overview

The [sequencer][g-sequencer] accepts L2 transactions from users. It is responsible for building blocks out of these. For
each such block, it also creates a corresponding [sequencer batch][g-sequencer-batch]. It is also responsible for
submitting each batch to a [data availability provider][g-avail-provider] (e.g. Ethereum calldata), which it does via
its [batcher][g-batcher] component.

The difference between an L2 block and a batch is subtle but important: the block includes an L2 state root, whereas the
batch only commits to transactions at a given L2 timestamp (equivalently: L2 block number). A block also includes a
reference to the previous block (\*).

(\*) This matters in some edge case where a L1 reorg would occur and a batch would be reposted to the L1 chain but not
the preceding batch, whereas the predecessor of an L2 block cannot possibly change.

This means that even if the sequencer applies a state transition incorrectly, the transactions in the batch will stil be
considered part of the canonical L2 chain. Batches are still subject to validity checks (i.e. they have to be encoded
correctly), and so are individual transactions within the batch (e.g. signatures have to be valid). Invalid batches and
invalid individual transactions within an otherwise valid batch are discarded by correct nodes.

If the sequencer applies a state transition incorrectly and posts an [output root][g-l2-output], then this output root
will be incorrect. The incorrect output root which will be challenged by a [fault proof][g-fault-proof], then replaced
by a correct output root **for the existing sequencer batches.**

Refer to the [Batch Submission specification][batcher-spec] for more information.

[batcher-spec]: batching.md

> **TODO** rewrite the batch submission specification
>
> Here are some things that should be included there:
>
> - There may be different concurrent data submissions to L1
> - There may be different actors that submit the data, the system cannot rely on a single EOA nonce value.
> - The batcher requests safe L2 safe head from the rollup node, then queries the execution engine for the block data.
>   - In the future we might be able to get the safe hea dinformation from the execution engine directly. Not possible
>     right now but there is an upstream geth PR open.

## Batch Submission Wire Format

[wire-format]: #batch-submission-wire-format

Batch submission is closely tied to L2 chain derivation because the derivation process must decode the batches that have
been encoded for the purpose of batch submission.

The [batcher][g-batcher] submits [batcher transactions][g-batcher-transaction] to a [data availability
provider][g-avail-provider]. These transactions contain one or multiple [channel frames][g-channel-frame], which are
chunks of data belonging to a [channel][g-channel].

A [channel][g-channel] is a sequence of [sequencer batches][g-sequencer-batch] (for sequential blocks) compressed
together. The reason to group multiple batches together is simply to obtain a better compression rate, hence reducing
data availability costs.

Channels might be too large to fit in a single [batcher transaction][g-batcher-transaction], hence we need to split it
into chunks known as [channel frames][g-channel-frame]. A single batcher transaction can also carry multiple frames
(belonging to the same or to different channels).

This design gives use the maximum flexibility in how we aggregate batches into channels, and split channels over batcher
transactions. It notably allows us to maximize data utilisation in a batcher transaction: for instance it allows us to
pack the final (small) frame of a window with large frames from the next window. It also allows the [batcher][g-batcher]
to employ multiple signers (private keys) to submit one or multiple channels in parallel (1).

(1) This helps alleviate issues where, because of transaction nonces, multiple transactions made by the same signer are
stuck waiting on the inclusion of a previous transaction.

Also note that we use a streaming compression scheme, and we do not need to know how many blocks a channel will end up
containing when we start a channel, or even as we send the first frames in the channel.

All of this is illustrated in the following diagram.

> **TODO** improve diagram
>
> - I'm a fan of the 4 lines "Transactions" to "L2 Blocks"
>   - albeit it would good to show that channels & frames can occur out of order
>   - but maybe that makes the diagram too hard to read and we can just include a comment afterwards saying that in
>       general, reordering is possible (maybe show a second diagram showcasing a simple reordering?
> - I think L1 blocks should be a new line above "Transactions" — also show deposits (w/ a number) for each block
> - We shouldn't use the L1 attributes tx as a separate line, it makes it look like a additional layer of "data
>   derivation", which it is not. Instead I would tag each L2 block with its epoch number.
> - Include numbered deposits under L2 blocks
> - Let's use a sequencing window of size 2 to keep the diagram small
> - Include a textual explanation of the diagram below it

![batch derivation chain diagram](./assets/batch-deriv-chain.svg)

### Batcher Transaction Format

Batcher transactions are encoded as `version_byte ++ rollup_payload` (where `++` denotes concatenation).

| `version_byte` | `rollup_payload`                               |
|----------------|------------------------------------------------|
| 0              | `frame ...` (one or more frames, concatenated) |

Unknown versions make the batcher transaction invalid (it must be ignored by the rollup node).

The `rollup_payload` may be right-padded with 0s, which will be ignored. It's allowed for them to be
interpreted as frames for channel 0, which must always be ignored.

> **TODO** specify batcher authentication (i.e. where do we store / make available the public keys of authorize batcher
> signers)

### Frame Format

A [channel frame][g-channel-frame] is encoded as:

```text
frame = channel_id ++ frame_number ++ frame_data_length ++ frame_data ++ is_last

channel_id        = random ++ timestamp
random            = bytes32
timestamp         = uvarint
frame_number      = uvarint
frame_data_length = uvarint
frame_data        = bytes
is_last           = bool
```

> **TODO** replace `uvarint` by fixed size integers

where:

- `uvarint` is a variable-length encoding of a 64-bit unsigned integer into between 1 and 9 bytes, [as specified in
  SQLite 4][sqlite-uvarint].
- `channel_id` uniquely identifies a channel as the concatenation of a random value and a timestamp
  - `random` is a random value such that two channels with different batches should have a different random value
  - `timestamp` is the time at which the channel was created (UNIX time in seconds)
  - The ID includes both the random value and the timestamp, in order to prevent a malicious sequencer from reusing
      the random value after the channel has [timed out][g-channel-timeout] (refer to the [batcher
      specification][batcher-spec] to learn more about channel timeouts). This will also allow us substitute `random` by
      a hash commitment to the batches, should we want to do so in the future.
  - Channels whose timestamp are higher than that of the L1 block they first appear in must be ignored. Note that L1
      nodes have a soft constraint to ignore blocks whose timestamps that are ahead of the wallclock time by a certain
      margin. (A soft constraint is not a consensus rule — nodes will accept such blocks in the canonical chain but will
      not attempt to build directly on them.)
- `frame_number` identifies the index of the frame within the channel
- `frame_data_length` is the length of `frame_data` in bytes
- `frame_data` is a sequence of bytes belonging to the channel, logically after the bytes from the previous frames
- `is_last` is a single byte with a value of 1 if the frame is the last in the channel, 0 if there are frames in the
  channel. Any other value makes the frame invalid (it must be ignored by the rollup node).

> **TODO**
>
> - Is that requirement to drop channels correct?
> - Is it implemented as such?
> - Do we drop the channel or just the first frame? End result is the same but this changes the channel bank size, which
>   can influence things down the line!!

[sqlite-uvarint]: https://www.sqlite.org/src4/doc/trunk/www/varint.wiki
[batcher-spec]: batching.md

### Channel Format

A channel is encoded as `channel_encoding`, defined as:

```text
rlp_batches = []
for batch in batches:
    rlp_batches.append(batch)
channel_encoding = compress(rlp_batches)
```

where:

- `batches` is the input, a sequence of batches byte-encoded as per the next section ("Batch Encoding")
- `rlp_batches` is the concatenation of the RLP-encoded batches
- `compress` is a function performing compression, using the ZLIB algorithm (as specified in [RFC-1950][rfc1950]) with
  no dictionary
- `channel_encoding` is the compressed version of `rlp_batches`

[rfc1950]: https://www.rfc-editor.org/rfc/rfc1950.html

When decompressing a channel, we limit the amount of decompressed data to `MAX_RLP_BYTES_PER_CHANNEL`, in order to avoid
"zip-bomb" types of attack (where a small compressed input decompresses to a humongous amount of data). If the
decompressed data exceeds the limit, things proceeds as thought the channel contained only the first
`MAX_RLP_BYTES_PER_CHANNEL` decompressed bytes.

> **TODO** specify `MAX_RLP_BYTES_PER_CHANNEL`

While the above pseudocode implies that all batches are known in advance, it is possible to perform streaming
compression and decompression of RLP-encoded batches. This means it is possible to start including channel frames in a
[batcher transaction][g-batcher-transaction] before we know how many batches (and how many frames) the channel will
contain.

### Batch Format

[batch-format]: #batch-format

Recall that a batch contains a list of transactions to be included in a specific L2 block.

A batch is encoded as `batch_version ++ content`, where `content` depends on the version:

| `batch_version` | `content`                                                             |
| --------------- | --------------------------------------------------------------------- |
| 0               | `rlp_encode([epoch_number, epoch_hash, timestamp, transaction_list])` |

where:

- `rlp_encode` is a function that encodes a batch according to the [RLP format], and `[x, y, z]` denotes a list
  containing items `x`, `y` and `z`
- `epoch_number` and `epoch_hash` are the number and hash of the L1 block corresponding to the [sequencing
  epoch][g-sequencing-epoch] of the L2 block
- `timestamp` is the timestamp of the L2 block
- `transaction_list` is an RLP-encoded list of [EIP-2718] encoded transactions.

[RLP format]: https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

Unknown versions make the batch invalid (it must be ignored by the rollup node), as do malformed contents.

The `epoch_number` and the `timestamp` must also respect the constraints listed in the [Batch
Buffering][batch-buffering] section, otherwise the batch is considered invalid.

------------------------------------------------------------------------------------------------------------------------

# Architecture

[architecture]: #architecture

The above describes the general process of L2 chain derivation, and specifies how batches are encoded within [batcher
transactions][g-batcher-transaction].

However, there remains many details to specify. These are mostly tied to the rollup node architecture for derivation.
Therefore we present this architecture as a way to specify these details.

A validator that only reads from L1 (and so doesn't interact with the sequencer directly) does not need to be
implemented in the way presented below. It does however need to derive the same blocks (i.e. it needs to be semantically
equivalent). We do believe the architecture presented below has many advantages.

## L2 Chain Derivation Pipeline

[pipeline]: #l2-chain-derivation-pipeline

Our architecture decomposes the derivation process into a pipeline made up of the following stages:

1. L1 Traversal
2. L1 Retrieval
3. Channel Bank
4. Batch Decoding (called `ChannelInReader` in the code)
5. Batch Buffering (Called `BatchQueue` in the code)
6. Payload Attributes Derivation (called `AttributesQueue` in the code)
7. Engine Queue

> **TODO** can we change code names for these three things? maybe as part of a refactor

The data flows flows from the start (outer) of the pipeline towards the end (inner). Each stage is able to push data to
the next stage.

However, data is *processed* in reverse order. Meaning that if there is any data to be processed in the last stage, it
will be processed first. Processing proceeds in "steps" that can be taken at each stage. We try to take as many steps as
possible in the last (most inner) stage before taking any steps in its outer stage, etc.

This ensures that we use the data we already have before pulling more data and minimizes the latency of data traversing
the derivation pipeline.

Each stage can maintain its own inner state as necessary. **In particular, each stage maintains a L1 block reference
(number + hash) to the latest L1 block such that all data originating from previous blocks has been processed, and the
data from that block is or has been processed.**

Let's briefly describe each stage of the pipeline.

### L1 Traversal

In the *L1 Traversal* stage, we simply read the header of the next L1 block. In normal operations, these will be new
L1 blocks as they get created, though we can also read old blocks while syncing, or in case of an L1 [re-org][g-reorg].

### L1 Retrieval

In the *L1 Retrieval* stage, we read the block we get from the outer stage (L1 traversal), and extract data for it. In
particular we extract a byte string that corresponds to the concatenation of the data in all the [batcher
transaction][g-batcher-transaction] belonging to the block. This byte stream encodes a stream of [channel
frames][g-channel-frame] (see the [Batch Submission Wire Format][wire-format] section for more info).

This frames are parsed, then grouped per [channel][g-channel] into a structure we call the *channel bank*.

Some frames are ignored:

- Frames where `frame.frame_number <= highest_frame_number`, where `highest_frame_number` is the highest frame number
  that was previously encountered for this channel.
  - i.e. in case of duplicate frame, the first frame read from L1 is considered canonical.
- Frames with a higher number than that of the final frame of the channel (i.e. the first frame marked with
  `frame.is_last == 1`) are ignored.
  - These frames could still be written into the channel bank if we haven't seen the final frame yet. But they will
      never be read out from the channel bank.

### Channel Bank

The *Channel Bank* stage is responsible from reading from the channel bank that was written to by the L1 retrieval
stage, and decompressing batches from these frames.

In principle, we should be able to read any channel that has any number of sequential frames at the "front" of the
channel (i.e. right after any frames that have been read from the bank already) and decompress batches from them. (Note
that if we did this, we'd need to keep partially decompressed batches around.)

However, our current implementation doesn't support streaming decompression, so currently we have to wait until either:

- We have received all frames in the channel (i.e. we received the last frame in the channel (`is_last == 1`) and every
  frame with a lower number).
- The channel has timed out (in which we case we read all contiguous sequential frames from the start of the channel).
  - A channel is considered to be *timed out* if `currentL1Block.timestamp > channeld_id.timestamp + CHANNEL_TIMEOUT`.
    - where `currentL1Block` is the L1 block maintained by this stage, which is the most recent L1 block whose frames
          have been added to the channel bank.

> **TODO** There is currently `MAX_CHANNEL_BANK_SIZE`, a notion about the maximum amount of channels we can keep track
> of.
>
> - Is this a semantic detail (i.e. if the batcher opens too many frames, valid channels can be dropped?)
> - If so, I feel **very strongly** about changing this. This ties us very much to the current implementation.
> - And it doesn't feel necessary given the channel timeout - if DOS is an issue we can reduce the channel timeout.

> **TODO** The channel queue is a bit weird as implemented (blocks all other channels until the first channel is closed
> / timed out. Also unclear why we need to wait for channel closure. Maybe something to revisit?
>
> cf. slack discussion with Proto

### Batch Decoding

In the *Batch Decoding* stage, we decompress the frames we received in the last stage, then parse
[batches][g-sequencer-batch] from the decompressed byte stream.

### Batch Buffering

[batch-buffering]: #batch-buffering

During the *Batch Buffering* stage, we reorder batches by their timestamps. If batches are missing for some [time
slots][g-time-slot] and a valid batch with a higher timestamp exists, this stage also generates empty batches to fill
the gaps.

Batches are pushed to the next stage whenever there is one or more sequential batches directly following the timestamp
of the current [safe L2 head][g-safe-l2-head] (the last block that can be derived from the canonical L1 chain).

Note that the presence of any gaps in the batches derived from L1 means that this stage will need to buffer for a whole
[sequencing window][g-sequencing-window] before it can generate empty batches (because the missing batch(es) could have
data in the last L1 block of the window in the worst case).

We also ignore invalid batches, which do not satisfy one of the following constraints:

- The timestamp is aligned to the [block time][g-block-time]:
  `(batch.timestamp - genesis_l2_timestamp) % block_time == 0`
- The timestamp is within the allowed range: `min_l2_timestamp <= batch.timestamp < max_l2_timestamp`, where
  - all these values are denominated in seconds
  - `min_l2_timestamp = prev_l2_timestamp + l2_block_time`
    - `prev_l2_timestamp` is the timestamp of the previous L2 block: the last block of the previous epoch,
      or the L2 genesis block timestamp if there is no previous epoch.
    - `l2_block_time` is a configurable parameter of the time between L2 blocks (on Optimism, 2s)
  - `max_l2_timestamp = max(l1_timestamp + max_sequencer_drift, min_l2_timestamp + l2_block_time)`
    - `l1_timestamp` is the timestamp of the L1 block associated with the L2 block's epoch
    - `max_sequencer_drift` is the maximum amount of time an L2 block's timestamp is allowed to get ahead of the
       timestamp of its [L1 origin][g-l1-origin]
  - Note that we always have `min_l2_timestamp >= l1_timestamp`, i.e. a L2 block timestamp is always equal or ahead of
    the timestamp of its [L1 origin][g-l1-origin].
- The batch is the first batch with `batch.timestamp` in this sequencing window, i.e. one batch per L2 block number.
- The batch only contains sequenced transactions, i.e. it must NOT contain any [deposited-type transactions][
  g-deposit-tx-type].

> **TODO** specify `max_sequencer_drift`

### Payload Attributes Derivation

In the *Payload Attributes Derivation* stage, we convert the batches we get from the previous stage into instances of
the [`PayloadAttributes`][g-payload-attr] structure. Such a structure encodes the transactions that need to figure into
a block, as well as other block inputs (timestamp, fee recipient, etc). Payload attributes derivation is detailed in the
section [Deriving Payload Attributes section][deriving-payload-attr] below.

### Engine Queue

In the *Engine Queue* stage, the previously derived `PayloadAttributes` structures are buffered and sent to the
[execution engine][g-exec-engine] to be executed and converted into a proper L2 block.

The engine queue maintains references to two L2 blocks:

- The [safe L2 head][g-safe-l2-head]: everything up to and including this block can be fully derived from the
  canonical L1 chain.
- The [unsafe L2 head][g-unsafe-l2-head]: blocks between the safe and unsafe heads are [unsafe
  blocks][g-unsafe-l2-block] that have not been derived from L1. These blocks either come from sequencing (in sequencer
  mode) or from [unsafe sync][g-unsafe-sync] to the sequencer (in validator mode).

If the unsafe head is ahead of the safe head, then [consolidation][g-consolidation] is attempted.

During consolidation, we consider the oldest unsafe L2 block, i.e. the unsafe L2 block directly after the safe head. If
the payload attributes match this oldest unsafe L2 block, then that block can be considered "safe" and becomes the new
safe head.

In particular, the following fields of the payload attributes are checked for equality with the block:

- `parent_hash`
- `timestamp`
- `randao`
- `fee_recipient`
- `transactions_list` (first length, then equality of each of the encoded transactions)

If consolidation fails, the unsafe L2 head is reset to the safe L2 head.

If the safe and unsafe L2 heads are identical (whether because of failed consolidation or not), we send the block to the
execution engine to be converted into a proper L2 block, which becomes both the new L2 safe and unsafe heads.

Interaction with the execution engine via the execution engine API is detailed in the [Communication with the Execution
Engine][exec-engine-comm] section.

### Resetting the Pipeline

It is possible to reset the pipeline, for instance if we detect an L1 [re-org][g-reorg]. For more details on this, see
the [Handling L1 Re-Orgs][handling-reorgs] section.

------------------------------------------------------------------------------------------------------------------------

# Deriving Payload Attributes

[deriving-payload-attr]: #deriving-payload-attributes

For every L2 block we wish to create, we need to build [payload attributes][g-payload-attr],
represented by an [expanded version][expanded-payload] of the [`PayloadAttributesV1`][eth-payload] object,
which includes the additional `transactions` and `noTxPool` fields.

[expanded-payload]: exec-engine.md#extended-payloadattributesv1
[eth-payload]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#payloadattributesv1

## Deriving the Transaction List

For each such block, we start from a [sequencer batch][g-sequencer-batch] matching the target L2 block number. This
could potentially be an empty auto-generated batch, if the L1 chain did not include a batch for the target L2 block
number. [Remember][batch-format] the batch includes a [sequencing epoch][g-sequencing-epoch] number, an L2 timestamp,
and a transaction list.

This block is part of a [sequencing epoch][g-sequencing-epoch],
whose number matches that of an L1 block (its *[L1 origin][g-l1-origin]*).
This L1 block is used to derive L1 attributes and (for the first L2 block in the epoch) user deposits.

Therefore, a [`PayloadAttributesV1`][expanded-payload] object must include the following transactions:

- one or more [deposited transactions][g-deposited], of two kinds:
  - a single *[L1 attributes deposited transaction][g-l1-attr-deposit]*, derived from the L1 origin.
  - for the first L2 block in the epoch, zero or more *[user-deposited transactions][g-user-deposited]*, derived from
    the [receipts][g-receipts] of the L1 origin.
- zero or more *[sequenced transactions][g-sequencing]*: regular transactions signed by L2 users, included in the
  sequencer batch.

Transactions **must** appear in this order in the payload attributes.

The L1 attributes are read from the L1 block header, while deposits are read from the L1 block's [receipts][g-receipts].
Refer to the [**deposit contract specification**][deposit-contract-spec] for details on how deposits are encoded as log
entries.

[deposit-contract-spec]: deposits.md#deposit-contract

## Building Individual Payload Attributes

[payload attributes]: #building-individual-payload-attributes

After deriving the transaction list, the rollup node constructs a [`PayloadAttributesV1`][expanded-payload] as follows:

- `timestamp` is set to the batch's timestamp.
- `random` is set to the *random* `prev_randao` L1 block attribute.
- `suggestedFeeRecipient` is set to an address determined by the system.
- `transactions` is the array of the derived transactions: deposited transactions and sequenced transactions, all
  encoded with [EIP-2718].
- `noTxPool` is set to `true`, to use the exact above `transactions` list when constructing the block.

[expanded-payload]: exec-engine.md#extended-payloadattributesv1

> **TODO** specify Optimism mainnet fee recipient

------------------------------------------------------------------------------------------------------------------------

# WARNING: BELOW THIS LINE, THE SPEC HAS NOT BEEN REVIEWED AND MAY CONTAIN MISTAKES

We still expect that the explanations here should be pretty useful.

------------------------------------------------------------------------------------------------------------------------

# Communication with the Execution Engine

[exec-engine-comm]: #communication-with-the-execution-engine

The [engine queue] is responsible for interacting with the execution engine, sending it
[`PayloadAttributesV1`][expanded-payload] objects and receiving L2 block references as a result. This happens whenever
the [safe L2 head][g-safe-l2-head] and the [unsafe L2 head][g-unsafe-l2-head] are identical, either because [unsafe
block consolidation][g-consolidation] failed or because no [unsafe L2 blocks][g-unsafe-l2-block] were known in the first
place. This section explains how this happens.

> **Note**: This only describes interaction with the execution engine in the context of L2 chain derivation from L1. The
> sequencer also interacts with the engine when it needs to create new L2 blocks using L2 transactions submitted by
> users.

Let:

- `refL2` be the (hash of) the current [safe L2 head][g-unsafe-l2-head]
- `finalizedRef` be the (hash of) the [finalized L2 head][g-finalized-l2-head]: the highest L2 block that can be fully
  derived from *[finalized][finality]* L1 blocks — i.e. L1 blocks older than two L1 epochs (64 L1 [time
  slots][g-time-slot]).
- `payloadAttributes` be some previously derived [payload attributes][g-payload-attr] for the L2 block with number
  `l2Number(refL2) + 1`

[finality]: https://hackmd.io/@prysmaticlabs/finality

Then we can apply the following pseudocode logic to update the state of both the rollup driver and execution engine:

```javascript
// request a new execution payload
forkChoiceState = {
    headBlockHash: refL2,
    safeBlockHash: refL2,
    finalizedBlockHash: finalizedRef,
}
[status, payloadID, rpcErr] = engine_forkchoiceUpdatedV1(forkChoiceState, payloadAttributes)
if (rpcErr != null) soft_error()
if (status != "VALID") payload_error()

// retrieve and execute the execution payload
[executionPayload, rpcErr] = engine_getPayloadV1(payloadID)
if (rpcErr != null) soft_error()

[status, rpcErr] = engine_newPayloadV1(executionPayload)
if (rpcErr != null) soft_error()
if (status != "VALID") payload_error()

refL2 = l2Hash(executionPayload)

// update head to new refL2
forkChoiceState = {
    headBlockHash: refL2,
    safeBlockHash: refL2,
    finalizedBlockHash: finalizedRef,
}
[status, payloadID, rpcErr] = engine_forkchoiceUpdatedV1(forkChoiceState, null)
if (rpcErr != null) soft_error()
if (status != "SUCCESS") payload_error()
```

As should apparent from the assignations, within the `forkChoiceState` object, the properties have the following
meaning:

- `headBlockHash`: block hash of the last block of the L2 chain, according to the sequencer.
- `safeBlockHash`: same as `headBlockHash`.
- `finalizedBlockHash`: the hash of the L2 block that can be fully derived from finalized L1 data, making it impossible
  to derive anything else.

Error handling:

- A `payload_error()` means the inputs were wrong, and the payload attributes should thus be dropped from the queue, and
  not reattempted.
- A `soft_error()` means that the interaction failed by chance, and should be reattempted.
- If the function completes without error, the attributes were applied successfully,
  and can be dropped from the queue while the tracked "safe head" is updated.

> **TODO** `finalizedRef` is not being changed yet, but can be set to point to a L2 block fully derived from data up to
> a finalized L1 block.

The following JSON-RPC methods are part of the [execution engine API][exec-engine]:

[exec-engine]: exec-engine.md

- [`engine_forkchoiceUpdatedV1`] — updates the forkchoice (i.e. the chain head) to `headBlockHash` if different, and
  instructs the engine to start building an execution payload given payload attributes the second argument isn't `null`
- [`engine_getPayloadV1`] — retrieves a previously requested execution payload
- [`engine_newPayloadV1`] — executes an execution payload to create a block

[`engine_forkchoiceUpdatedV1`]: exec-engine.md#engine_forkchoiceUpdatedV1
[`engine_getPayloadV1`]: exec-engine.md#engine_newPayloadV1
[`engine_newPayloadV1`]: exec-engine.md#engine_newPayloadV1

The execution payload is an object of type [`ExecutionPayloadV1`][eth-payload].

[eth-payload]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1

------------------------------------------------------------------------------------------------------------------------

# Handling L1 Re-Orgs

[handling-reorgs]: #handling-l1-re-orgs

The [L2 chain derivation pipeline][pipeline] as described above assumes linear progression of the L1 chain.

If the L1 chain [re-orgs][g-reorg], the rollup node must re-derive sections of the L2 chain such that it derives the
same L2 chain that a rollup node would derive if it only followed the new L1 chain.

A re-org can be recovered without re-deriving the full L2 chain, by resetting each pipeline stage from end (Engine
Queue) to start (L1 Traversal).

The general idea is to backpropagate the new L1 head through the stages, and reset the state in each stage so that the
stage will next process data originating from that block onwards.

## Resetting the Engine Queue

The engine queue maintains references to two L2 blocks:

- The safe L2 block (or *safe head*): everything up to and including this block can be fully derived from the
  canonical L1 chain.
- The unsafe L2 block (or *unsafe head*): blocks between the safe and unsafe heads are blocks that have not been
  derived from L1. These blocks either come from sequencing (in sequencer mode) or from "unsafe sync" to the sequencer
  (in validator mode).

When resetting the L1 head, we need to rollback the safe head such that the L1 origin of the new safe head is a
canonical L1 block (i.e. an the new L1 head, or one of its ancestors). We achieved this by walking back the L2 chain
(starting from the current safe head) until we find such an L2 block. While doing this, we must take care not to walk
past the [L2 genesis][g-l2-genesis] or L1 genesis.

The unsafe head does not necessarily need to be reset, as long as its L1 origin is *plausible*. The L1 origin of the
unsafe head is considered plausible as long as it is in the canonical L1 chain or is ahead (higher number) than the head
of the L1 chain. When we determine that this is no longer the case, we reset the unsafe head to be equal to the safe
head.

> **TODO** Don't we always need to discard the unsafe head when there is a L1 re-org, because the unsafe head's origin
> builds on L1 blocks that have been re-orged away?
>
> I'm guessing maybe we received some unsafe blocks that build upon the re-orged L2, which we accept without relating
> them back to the safe head?

## Resetting Payload Attribute Derivation

In payload attribute derivation, we need to ensure that the L1 head is reset to the safe L2 head's L1 origin. In the
worst case, this would be as far back as `SWS` ([sequencing window][g-sequencing-window] size) blocks before the engine
queue's L1 head.

In the worst case, a whole sequencing window of L1 blocks was required to derive the L2 safe head (meaning that
`safeL2Head.l1Origin == engineQueue.l1Head - SWS`). This means that to derive the next L2 block, we have to read data
derived from L1 block `engineQueue.l1Head - SWS` and onwards, hence the need to reset the L1 head back to that value for
this stage.

However, in general, it is only necessary to reset as far back as `safeL2Head.l1Origin`, since it marks the start of the
sequencing window for the safe L2 head's epoch. As such, the next L2 block never depends on data derived from L1 blocks
before `safeL2Head.l1Origin`.

> **TODO** in the implementation, we always rollback by SWS, which is unecessary
> Quote from original spec:"We must find the first L2 block whose complete sequencing window is unchanged in the reorg."

> **TODO** sanity check this section, it was incorrect in previous spec, and confused me multiple times

## Resetting Batch Decoding

The batch decoding stage is simply reset by resetting its L1 head to the payload attribute derivation stage's L1 head.
(The same reasoning as the payload derivation stage applies.)

## Resetting Channel Buffering

> **Note**: in this section, the term *next (L2) block* will refer to the block that will become the next L2 safe head.

> **TODO** The above can be changed in the case where we always reset the unsafe head to the safe head upon L1 re-org.
> (See TODO above in "Resetting the Engine Queue")

Because we group [sequencer batches][g-sequencer-batch] into [channels][g-channel], it means that decoding a batch that
has data posted (in a [channel frame][g-channel-frame]) within the sequencing window of its epoch might require [channel
frames][g-channel-frame] posted before the start of the [sequencing window][g-sequencing-window]. Note that this is only
possible if we start sending channel frames before knowing all the batches that will go into the channel.

In the worst case, decoding the batch for the next L2 block would require reading the last frame from a channel, posted
in a [batcher transaction][g-batcher-transaction] in `safeL2Head.l1Origin + 1` (second L1 block of the next L2 block's
epoch sequencing window, assuming it is in the same epoch as `safeL2Head`).

> **Note**: In reality, there are no checks or constraints preventing the batch from landing in `safeL2Head.l1Origin`.
> However this would be strange, because the next L2 block is built after the current L2 safe block, which requires
> reading the deposits L1 attributes and deposits from `safeL2Head.l1Origin`. Still, a wonky or misbehaving sequencer
> could post a batch for the L2 block `safeL2Head + 1` on L1 block `safeL2Head.1Origin`.

Keeping things worst case, `safeL2Head.l1Origin` would also be the last allowable block for the frame to land. The
allowed time range for frames within a channel to land on L1 is `[channel_id.timestamp, channel_id.timestamp +
CHANNEL_TIMEOUT]`. The allowed L1 block range for these frames are any L1 block whose timestamp falls inside this time
range.

Therefore, to be safe, we can reset the L1 head of Channel Buffering to the oldest L1 block whose timestamp is higher
than `safeL2Head.l1Origin.timestamp - CHANNEL_TIMEOUT`.

> **Note**: The above is what the implementation currently does.

In reality it's only strictly necessary to reset the oldest L1 block whose timestamp is higher than the oldest
`channel_id.timestamp` found in the batcher transaction that is not older than `safeL2Head.l1Origin.timestamp -
CHANNEL_TIMEOUT`.

We define `CHANNEL_TIMEOUT = 600`, i.e. 10 hours.

> **TODO** does `CHANNEL_TIMEOUT` have a relationship with `SWS`?
>
> I think yes, it has to be shorter than `SWS` but ONLY if we can't do streaming decryption (the case currently).
> Otherwise it could be shorter or longer.

— and explain its relationship with `SWS` if any?

This situation is the main purpose of the [channel timeout][g-channel-timeout]: without the timeout, we might have to
look arbitrarily far back on L1 to be able to decompress batches, which is not acceptable for performance reasons.

The other puprose of the channel timeout is to avoid having the rollup node keep old unclosed channel data around
forever.

Once the L1 head is reset, we then need to discard any frames read from blocks more recent than this updated L1 head.

## Resetting L1 Retrieval & L1 Traversal

These are simply reset by resetting their L1 head to `channelBuffering.l1Head`, and dropping any buffered data.

## Reorgs Post-Merge

Note that post-[merge], the depth of re-orgs will be bounded by the [L1 finality delay][l1-finality] (every 2 epochs, or
approximately 12 minutes, unless an attacker controls more than 1/3 of the total stake).

[merge]: https://ethereum.org/en/upgrades/merge/
[l1-finality]: https://ethereum.org/en/developers/docs/consensus-mechanisms/pos/#finality

> **TODO** This was in the spec:
>
> In practice, we'll pick an already-finalized L1 block as L2
> inception point to preclude the possibility of a re-org past genesis, at the cost of a few empty blocks at the start
> of the L2 chain.
>
> This makes sense, but is in conflict with how the [L2 chain inception][g-l2-chain-inception] is currently determined,
> which is via the L2 output oracle deployment & upgrades.
