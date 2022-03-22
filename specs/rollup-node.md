# Rollup Node Specification

<!-- All glossary references in this file. -->
[g-rollup-node]: glossary.md#rollup-node
[g-derivation]: glossary.md#L2-chain-derivation
[g-payload-attr]: glossary.md#payload-attributes
[g-block]: glossary.md#block
[g-exec-engine]: glossary.md#execution-engine
[g-reorg]: glossary.md#re-organization
[g-rollup-driver]: glossary.md#rollup-driver
[g-inception]: glossary.md#L2-chain-inception
[g-receipts]: glossary.md#receipt
[g-deposit-contract]: glossary.md#deposit-contract
[g-deposits]: glossary.md#deposits
[g-deposited]: glossary.md#deposited-transaction
[g-l1-attr-deposit]: glossary.md#l1-attributes-deposited-transaction
[g-user-deposited]: glossary.md#user-deposited-transaction
[g-l1-attr-predeploy]: glossary.md#l1-attributes-predeployed-contract
[g-depositing-call]: glossary.md#depositing-call
[g-depositing-transaction]: glossary.md#depositing-transaction
[g-mpt]: glossary.md#merkle-patricia-trie
[g-sequencing-window]: glossary.md#sequencing-window
[g-sequencing]: glossary.md#sequencing
[g-sequencer-batch]: glossary.md#sequencer-batch

The [rollup node][g-rollup-node] is the component responsible for [deriving the L2 chain][g-derivation] from L1 blocks
(and their associated [receipts][g-receipts]). This process happens in three steps:

1. Select a [sequencing window][g-sequencing-window] from the L1 chain, on top of the last L2 block:
   a list of blocks, with transactions and associated receipts.
2. Read L1 information, deposits, and sequencing batches in order to generate [payload attributes][g-payload-attr]
   (essentially [a block without output properties][g-block]).
3. Pass the payload attributes to the [execution engine][g-exec-engine], so that the L2 block (including [output block
   properties][g-block]) may be computed.

While this process is conceptually a pure function from the L1 chain to the L2 chain, it is in practice incremental. The
L2 chain is extended whenever new L1 blocks are added to the L1 chain. Similarly, the L2 chain re-organizes whenever the
L1 chain [re-organizes][g-reorg].

The part of the rollup node that derives the L2 chain is called the [rollup driver][g-rollup-driver]. This document is
currently only concerned with the specification of the rollup driver.

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [L2 Chain Derivation](#l2-chain-derivation)
  - [From L1 chain to Sequencing Window](#from-l1-chain-to-sequencing-window)
  - [From L1 Blocks to Payload Attributes](#from-l1-blocks-to-payload-attributes)
    - [Reading L1 inputs](#reading-l1-inputs)
    - [Encoding the L1 Attributes Deposited Transaction](#encoding-the-l1-attributes-deposited-transaction)
    - [Encoding User-Deposited Transactions](#encoding-user-deposited-transactions)
    - [Building the Payload Attributes](#building-the-payload-attributes)
  - [From Payload Attributes to L2 Block](#from-payload-attributes-to-l2-block)
    - [Inductive Derivation Step](#inductive-derivation-step)
    - [Engine API Error Handling](#engine-api-error-handling)
    - [Finalization Guarantees](#finalization-guarantees)
  - [Whole L2 Chain Derivation](#whole-l2-chain-derivation)
  - [L2 Output RPC method](#l2-output-rpc-method)
    - [Output Method API](#output-method-api)
- [Handling L1 Re-Orgs](#handling-l1-re-orgs)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# L2 Chain Derivation

[l2-chain-derivation]: #l2-chain-derivation

This section specifies how the [rollup driver][g-rollup-driver] derives a sequence of L2 blocks per sequencing window.

Every L2 block carries transactions of two categories:

- *[deposited transactions][g-deposited]*: two kinds:
  - derived from the L1 chain: a single *[L1 attributes deposited transaction][g-l1-attr-deposit]* (always first).
  - derived from [receipts][g-receipts]: zero or more *[user-deposited transactions][g-user-deposited]*.
- *[sequenced transactions][g-sequencing]*: derived from [sequencer batches][g-sequencer-batch],
  zero or more regular transactions, signed by L2 users.

------------------------------------------------------------------------------------------------------------------------

## From L1 chain to Sequencing Window

A [sequencing window][g-sequencing-window] is a fixed number consecutive L1 blocks that a derivation step takes as
input. The window is identified by an `epoch`, equal to the block number of the first block in the window.

As the full derivation of the L2 chain by the driver progresses each derivation step shifts the window forward by a
single L1 block: the windows overlap.

Each sequencing window is derived into a variable number of L2 blocks, depending on the timestamps of L1 and L2.

The L2 has a fixed block time and no more than one batch per block,
meaning that gaps between the batches (ordered by timestamp) are interpreted as batches with empty transaction-lists,
thus construing L2 blocks that only contain deposit transaction(s).

The L2 blocks produced by a sequencing window are bounded by timestamp:

- `min_l2_timestamp = prev_l2_timestamp + l2_block_time`
- `max_l2_timestamp = l1_timestamp + l2_block_time`, where `l1_timestamp` is the timestamp of the
  first L1 block of the sequencing window. (maximum bound, may not be aligned with block time)

If there are no batches present in the sequencing window then the L2 chain is extended up to `max_l2_timestamp` (incl.)
with empty batches, but otherwise regular block derivation.

Note that with short block times on L1 the L2 time may increment beyond the L1 time,
but the longer target block time of L1 will correct back and allow the timestamps to align again.

## From L1 Blocks to Payload Attributes

### Reading L1 inputs

The rollup reads the following data from the [sequencing window][g-sequencing-window]:

- Of the *first* block in the window only:
  - L1 block attributes:
    - block number
    - timestamp
    - basefee
    - *random* (the output of the [`RANDOM` opcode][random])
  - L1 log entries emitted for [user deposits][g-deposits], derived transactions are augmented with
    `blockHeight` and `transactionIndex` of the transaction in L2.
- Of each block in the window:
  - Sequencer batches, derived from the transactions:
    - The transaction receiver is the sequencer inbox address
    - The transaction must be signed by a recognized sequencer account
    - The calldata may contain a bundle of batches. *(calldata will be substituted with blob data in the future.)*
    - Batches not matching filter criteria are ignored:
      - `batch.epoch == sequencing_window.epoch`, i.e. for this sequencing window
      - `(batch.timestamp - genesis_l2_timestamp) % block_time == 0`, i.e. timestamp is aligned
      - `min_l2_timestamp < batch.timestamp < max_l2_timestamp`, i.e. timestamp is within range
      - The batch is the first batch with `batch.timestamp` in this sequencing window,
        i.e. one batch per L2 block number
      - The batch only contains sequenced transactions, i.e. it must NOT contain any Deposit-type transactions.

[random]: https://eips.ethereum.org/EIPS/eip-4399

A bundle of batches is versioned by prefixing with a bundle version byte: `bundle = bundle_version ++ bundle_data`.

Bundle versions:

- `0`: `bundle_data = RLP([batch_0, batch_1, ..., batch_N])`
- `1`: `bundle_data = compress(RLP([batch_0, batch_1, ..., batch_N]))` (compression algorithm TBD)

A batch is also versioned by prefixing with a version byte: `batch = batch_version ++ batch_data`
and encoded as a byte-string (including version prefix byte) in the bundle RLP list.

Batch versions:

- `0`: `batch_data = RLP([epoch, timestamp, transaction_list])`, where each

Batch contents:

- `epoch` is the sequencing window epoch, i.e. the first L1 block number
- `timestamp` is the L2 timestamp of the block
- `transaction_list` is an RLP encoded list of [EIP-2718] encoded transactions.

[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

The L1 attributes are read from the L1 block header, while deposits are read from the block's [receipts][g-receipts].
Refer to the [**deposit contract specification**][deposit-contract-spec] for details on how deposits are encoded as log
entries.

[deposit-contract-spec]: deposits.md#deposit-contract

Each of the derived `PayloadAttributes` starts with a L1 Attributes transaction.
Like other derived deposits, this does not have to be batch-submitted, and exposes the required L1 information for the
process of finding the sync starting point of the L2 chain, without requiring L2 state access.

The [User-deposited] transactions are all put in the first of the derived `PayloadAttributes`,
inserted after the L1 Attributes transaction, before any [sequenced][g-sequencing] transactions.

### Encoding the L1 Attributes Deposited Transaction

The [L1 attributes deposited transaction][g-l1-attr-deposit] is a call that submits the L1 block attributes (listed
above) to the [L1 attributes predeployed contract][g-l1-attr-predeploy].

To encode the L1 attributes deposited transaction, refer to the following sections of the deposits spec:

- [The Deposited Transaction Type](deposits.md#the-deposited-transaction-type)
- [L1 Attributes Deposited Transaction](deposits.md#l1-attributes-deposited-transaction)

### Encoding User-Deposited Transactions

A [user-deposited-transactions][g-deposited] is an L2 transaction derived from a [user deposit][g-deposits] submitted on
L1 to the [deposit contract][g-deposit-contract]. Refer to the [deposit contract specification][deposit-contract-spec]
for more details.

The user-deposited transaction is derived from the log entry emitted by the [depositing call][g-depositing-call], which
is stored in the [depositing transaction][g-depositing-transaction]'s log receipt.

To encode user-deposited transactions, refer to the following sections of the deposits spec:

- [The Deposited Transaction Type](deposits.md#the-deposited-transaction-type)
- [User-Deposited Transactions](deposits.md#user-deposited-transactions)

### Building the Payload Attributes

[payload attributes]: #building-the-payload-attributes

From the data read from L1 and the encoded transactions, the rollup node constructs the [payload
attributes][g-payload-attr] as an [expanded version][expanded-payload] of the [`PayloadAttributesV1`] object, which
includes an additional `transactions` field.

The object's properties must be set as follows:

- `timestamp` is set to the timestamp of the L1 block.
- `random` is set to the *random* L1 block attribute
- `suggestedFeeRecipient` is set to an address determined by the system
- `transactions` is an array of the derived transactions: deposited transactions and sequenced transactions.
   All encoded with [EIP-2718]. Sequenced transactions must exclude any Deposit-type transactions.

[expanded-payload]: exec-engine.md#extended-payloadattributesv1
[`PayloadAttributesV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#payloadattributesv1

------------------------------------------------------------------------------------------------------------------------

## From Payload Attributes to L2 Block

Once the [payload attributes] for a given L1 block `B` have been built, and if we have already derived an L2 block from
`B`'s parent block, then we can use the payload attributes to derive a new L2 block.

### Inductive Derivation Step

Let

- `refL2` be the (hash of) the current L2 chain head
- `refL1` be the (hash of) the L1 block from which `refL2` was derived
- `payloadAttributes` be some previously derived [payload attributes] for the L1 block with number `l1Number(refL1) + 1`

Then we can apply the following pseudocode logic to update the state of both the rollup driver and execution engine:

```javascript
// request a new execution payload
forkChoiceState = {
    headBlockHash: refL2,
    safeBlockHash: refL2,
    finalizedBlockHash: l2BlockHashAt(l2Number(refL2) - FINALIZATION_DELAY_BLOCKS)
}
[status, payloadID] = engine_forkchoiceUpdatedV1(forkChoiceState, payloadAttributes)
if (status != "SUCCESS") error()

// retrieve and execute the execution payload
[executionPayload, error] = engine_getPayloadV1(payloadID)
if (error != null) error()
[status, latestValidHash, validationError] = engine_executePayloadV1(executionPayload)
if (status != "VALID" || validationError != null) error()

refL2 = latestValidHash
refL1 = l1HashForNumber(l1Number(refL1) + 1))

// update head to new refL2
forkChoiceState = {
    headBlockHash: refL2,
    safeBlockHash: refL2,
    finalizedBlockHash: l2BlockHashAt(l2Number(headBlockHash) - FINALIZATION_DELAY_BLOCKS)
}
[status, payloadID] = engine_forkchoiceUpdatedV1(refL2, null)
if (status != "SUCCESS") error()
```

The following JSON-RPC methods are part of the [execution engine API][exec-engine]:

> **TODO** fortify the execution engine spec with more information regarding JSON-RPC, notably covering
> information found [here][json-rpc-info-1] and [here][json-rpc-info-2]

[json-rpc-info-1]: https://github.com/ethereum-optimism/optimistic-specs/blob/a3ffa9a8c825d155a0469659b3101db5f41eecc4/specs/rollup-node.md#from-l1-blocks-to-payload-attributes
[json-rpc-info-2]: https://github.com/ethereum-optimism/optimistic-specs/blob/a3ffa9a8c825d155a0469659b3101db5f41eecc4/specs/rollup-node.md#building-the-l2-block-with-the-execution-engine

[exec-engine]: exec-engine.md

- [`engine_forkchoiceUpdatedV1`] — updates the forkchoice (i.e. the chain head) to `headBlockHash` if different, and
  instructs the engine to start building an execution payload given payload attributes the second argument isn't `null`
- [`engine_getPayloadV1`] — retrieves a previously requested execution payload
- [`engine_executePayloadV1`] — executes an execution payload to create a block

[`engine_forkchoiceUpdatedV1`]: exec-engine.md#engine_forkchoiceUpdatedV1
[`engine_getPayloadV1`]: exec-engine.md#engine_executepayloadv1
[`engine_executePayloadV1`]: exec-engine.md#engine_executepayloadv1

The execution payload is an object of type [`ExecutionPayloadV1`].

[`ExecutionPayloadV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1

Within the `forkChoiceState` object, the properties have the following meaning:

- `headBlockHash`: block hash of the last block of the L2 chain, according to the rollup driver.
- `safeBlockHash`: same as `headBlockHash`.
- `finalizedBlockHash`: the hash of the block whose number is `l2Number(headBlockHash) - FINALIZATION_DELAY_BLOCKS` if
  the number of that block is `>= L2_CHAIN_INCEPTION`, 0 otherwise (\*) See the [Finalization Guarantees][finalization]
  section for more details.

(\*) where:

- `FINALIZATION_DELAY_BLOCKS == 50400` (approximately 7 days worth of L1 blocks)
- `L2_CHAIN_INCEPTION` is the [L2 chain inception][g-inception] (the number of the first L1 block for which an L2 block
  was produced).

Finally, the `error()` function signals an error that must be handled by the implementation. Refer to the next section
for more details.

### Engine API Error Handling

[error-handling]: #engine-api-error-handling

All invocations of [`engine_forkchoiceUpdatedV1`], [`engine_getPayloadV1`] and [`engine_executePayloadV1`] by the
rollup driver should not result in errors assuming conformity with the specification. Said otherwise, all errors are
implementation concerns and it is up to them to handle them (e.g. by retrying, or by stopping the chain derivation and
requiring manual user intervention).

The following scenarios are assimilated to errors:

- [`engine_forkchoiceUpdatedV1`] returning a `status` of `"SYNCING"` instead of `"SUCCESS"` whenever passed a
  `headBlockHash` that it retrieved from a previous call to [`engine_executePayloadV1`].
- [`engine_executePayloadV1`] returning a `status` of `"SYNCING"` or `"INVALID"` whenever passed an execution payload
  that was obtained by a previous call to [`engine_getPayloadV1`].

### Finalization Guarantees

[finalization]: #finalization-guarantees

As stated earlier, an L2 block is considered *finalized* after a delay of `FINALIZATION_DELAY_BLOCKS == 50400` L1 blocks
after the L1 block that generated it. This is a duration of approximately 7 days worth of L1 blocks. This is also known
as the "fault proof window", as after this time the block can no longer be challenged by a fault proof.

L1 Ethereum reaches [finality][l1-finality] approximately every [12.8 minutes][consensus-time-params]. L2 blocks generated from finalized L1 blocks
are "safer" than most recent L2 blocks because they will never disappear from the chain's history because of a re-org.
However, they can still be challenged by a fault proof until the end of the fault proof window.

[l1-finality]: https://www.paradigm.xyz/2021/07/ethereum-reorgs-after-the-merge
[consensus-time-params]: https://github.com/ethereum/consensus-specs/blob/v1.0.0/specs/phase0/beacon-chain.md#time-parameters

> **TODO** the spec doesn't encode the notion of fault proof yet, revisit this (and include links) when it does

## Whole L2 Chain Derivation

The [block derivation](#from-l1-blocks-to-payload-attributes) presents an inductive process:
given that we know the last L2 block derived from the previous [sequencing window][g-sequencing-window], as well as the
next [sequencing window][g-sequencing-window], then we can derive [payload attributes] of the next L2 blocks.

To derive the whole L2 chain from scratch, we simply start with the L2 genesis block as the last L2 block, and the
block at height `L2_CHAIN_INCEPTION + 1` as the start of the next sequencing window.
Then we iteratively apply the derivation process from the previous section by shifting the sequencing window one L1
block forward each step, until there is an insufficient number of L1 blocks left for a complete sequencing window.

> **TODO** specify genesis block

## L2 Output RPC method

The Rollup node has its own RPC method, `optimism_outputAtBlock` which returns the
a 32 byte hash corresponding to the [SSZ] encoded [L2Output](./proposals.md#l2-output-commitment-construction).

[SSZ]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md

### Output Method API

The input and return types here are as defined by the [engine API specs][engine-structures]).

[engine-structures]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#structures

- method: `optimism_outputAtBlock`
- params:
  1. `QUANTITY` - L2 integer block number, or the strings `"safe"`, `"latest"`, or `"pending"`
- returns:
  1. `DATA` - The 32 byte output root

# Handling L1 Re-Orgs

[l1-reorgs]: #handling-L1-re-orgs

The [previous section on L2 chain derivation][l2-chain-derivation] assumes linear progression of the L1 chain. It is
also applicable for batch processing, meaning that any given point in time, the canonical L2 chain is given by
processing the whole L1 chain since the [L2 chain inception][g-inception].

> By itself, the previous section fully specifies the behaviour of the rollup driver. **The current section is
> non-specificative** but shows how L1 re-orgs can be handled in practice.

In practice, the L1 chain is processed incrementally. However, the L1 chain may occasionally [re-organize][g-reorg],
meaning the head of the L1 chain changes to a block that is not the child of the previous head but rather another
descendant of an ancestor of the previous head. In that case, the rollup driver must first search for the common L1
ancestor, and can re-derive the L2 chain from that L1 block and onwards.

The starting point of the re-derivation is a pair `(refL2, nextRefL1)` where `refL2` refers to the L2 block to build
upon and `nextRefL1` refers to the next L1 block to derive from (i.e. if `refL2` is derived from L1 block `refL1`,
`nextRefL1` is the canonical L1 block at height `l1Number(refL1) + 1`).

In practice, the happy path (no re-org) and the re-org paths are merged. The happy path is simply a special case of the
re-org path where the starting point of the re-derivation is `(currentL2Head, newL1Block)`.

After a `(currentL2Head, newL1Block)` starting point is found, derivation can continue when a complete sequencing window
of canonical L1 blocks following the starting point is retrieved.

This re-derivation starting point can be found by applying the following algorithm:

1. (Initialization) Set the initial `refL2` to the head block of the L2 execution engine.
2. Set `parentL2` to `refL2`'s parent block and `refL1` to the L1 block that `refL2` was derived from.
3. Fetch `currentL1`, the canonical L1 block at the same height as `refL1`.

- If `currentL1 == refL1`, then `refL2` was built on a canonical L1 block:
  - Find the next L1 block (it may not exist yet) and return `(refL2, nextRefL1)` as the starting point of the
    re-derivation.
    - It is necessary to ensure that no L1 re-org occurred during this lookup, i.e. that `nextRefL1.parent == refL1`.
    - If the next L1 block does not exist yet, there is no re-org, and nothing new to derive, and we can abort the
        process.
- Otherwise, if `refL2` is the L2 genesis block, we have re-orged past the genesis block, which is an error that
  requires a re-genesis of the L2 chain to fix (i.e. creating a new genesis configuration) (\*)
- Otherwise, if either `currentL1` does not exist, or `currentL1 != refL1`, set `refL2` to `parentL2` and restart this
  algorithm from step 2.
  - Note: if `currentL1` does not exist, it means we are in a re-org to a shorter L1 chain.
  - Note: as an optimization, we can cache `currentL1` and reuse it as the next value of `nextRefL1` to avoid an
        extra lookup.

Note that post-[merge], the depth of re-orgs will be bounded by the [L1 finality delay][l1-finality] (every 2 epochs,
approximately 12 minutes).

(\*) Post-merge, this is only possible for 12 minutes. In practice, we'll pick an already-finalized L1 block as L2
inception point to preclude the possibility of a re-org past genesis, at the cost of a few empty blocks at the start of
the L2 chain.

[merge]: https://ethereum.org/en/eth2/merge/
