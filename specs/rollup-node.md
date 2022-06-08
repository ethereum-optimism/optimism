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
  - [From L1 Sequencing Window to L2 Payload Attributes](#from-l1-sequencing-window-to-l2-payload-attributes)
    - [Reading L1 inputs](#reading-l1-inputs)
    - [Encoding the L1 Attributes Deposited Transaction](#encoding-the-l1-attributes-deposited-transaction)
    - [Encoding User-Deposited Transactions](#encoding-user-deposited-transactions)
    - [Deriving all Payload Attributes of a sequencing window](#deriving-all-payload-attributes-of-a-sequencing-window)
      - [Building individual Payload Attributes](#building-individual-payload-attributes)
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

## From L1 Sequencing Window to L2 Payload Attributes

A [sequencing window][g-sequencing-window] is a fixed number consecutive L1 blocks that a derivation step takes as
input. The window is identified by an `epoch`, equal to the block number of the first block in the window.

The derivation of the L2 chain from the L1 chain happens in steps.
Each step adds a variable number of L2 blocks to the L2 chain, derived from the sequencing window for the given epoch.
For epoch `N`, the sequencing window comprises L1 blocks `[N, N + SEQUENCING_WINDOW_SIZE)`.
Note that the sequencing windows overlap.

### Reading L1 inputs

The rollup reads the following data from the [sequencing window][g-sequencing-window]:

- Of the *first* block in the window only:
  - L1 block attributes:
    - block number
    - timestamp
    - basefee
    - *random* (the output of the [`RANDOM` opcode][random])
  - L1 log entries emitted for [user deposits][g-deposits], augmented with a [sourceHash](./deposits.md#).
- Of each block in the window:
  - Sequencer batches, derived from the transactions:
    - The transaction receiver is the sequencer inbox address
    - The transaction must be signed by a recognized sequencer account
    - The calldata may contain a bundle of batches. *(calldata will be substituted with blob data in the future.)*
    - Batches not matching filter criteria are ignored:
      - `batch.epoch == sequencing_window.epoch`, i.e. for this sequencing window
      - `(batch.timestamp - genesis_l2_timestamp) % block_time == 0`, i.e. timestamp is aligned
      - `min_l2_timestamp <= batch.timestamp < max_l2_timestamp`, i.e. timestamp is within range
        - `min_l2_timestamp = prev_l2_timestamp + l2_block_time`
          - `prev_l2_timestamp` is the timestamp of the previous L2 block: the last block of the previous epoch,
              or the L2 genesis block timestamp if there is no previous epoch.
          - `l2_block_time` is a configurable parameter of the time between L2 blocks
        - `max_l2_timestamp = max(l1_timestamp + max_sequencer_drift, min_l2_timestamp + l2_block_time)`
          - `l1_timestamp` is the timestamp of the L1 block associated with the L2 block's epoch
          - `max_sequencer_drift` is the most a sequencer is allowed to get ahead of L1
      - The batch is the first batch with `batch.timestamp` in this sequencing window,
        i.e. one batch per L2 block number.
      - The batch only contains sequenced transactions, i.e. it must NOT contain any Deposit-type transactions.

Note that after the above filtering `min_l2_timestamp >= l1_timestamp` always holds,
i.e. a L2 block timestamp is always equal or ahead of the timestamp of the corresponding L1 origin block.

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
entries. The deposited and sequenced transactions are combined when the Payload Attributes are constructed.

[deposit-contract-spec]: deposits.md#deposit-contract

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

### Deriving all Payload Attributes of a sequencing window

A sequencing window is derived into a variable number of L2 blocks, defined by a range of timestamps:

- Starting at `min_l2_timestamp`, as defined in the batch filtering.
- Up to and including
  `new_head_l2_timestamp = max(highest_valid_batch_timestamp, next_l1_timestamp - l2_block_time, min_l2_timestamp)`
  - `highest_valid_batch_timestamp = max(batch.timestamp for batch in filtered_batches)`,
    or `0` if no there are no `filtered_batches`.
    `batch.timestamp` refers to the L2 block timestamp encoded in the batch.
  - `next_l1_timestamp` is the timestamp of the next L1 block.

The L2 chain is extended to `new_head_l2_timestamp` with blocks at a fixed block time (`l2_block_time`).
This means that every `l2_block_time` that has no batch is interpreted as one with no sequenced transactions.

Each of the derived `PayloadAttributes` starts with a L1 Attributes transaction.
Like other derived deposits, this does not have to be batch-submitted, and exposes the required L1 information for the
process of finding the sync starting point of the L2 chain, without requiring L2 state access.

The [User-deposited] transactions are all put in the first of the derived `PayloadAttributes`,
inserted after the L1 Attributes transaction, before any [sequenced][g-sequencing] transactions.

#### Building individual Payload Attributes

[payload attributes]: #building-individual-payload-attributes

From the timestamped transaction lists derived from the sequencing window, the rollup node constructs [payload
attributes][g-payload-attr] as an [expanded version][expanded-payload] of the [`PayloadAttributesV1`] object, which
includes the additional `transactions` and `noTxPool` fields.

Each of the timestamped transaction lists translates to a `PayloadAttributesV1` as follows:

- `timestamp` is set to the timestamp of the transaction list.
- `random` is set to the *random* `execution_payload.prev_randao` L1 block attribute
- `suggestedFeeRecipient` is set to an address determined by the system
- `transactions` is the array of the derived transactions: deposited transactions and sequenced transactions.
  All encoded with [EIP-2718].
- `noTxPool` is set to `true`, to use the exact above `transactions` list when constructing the block.

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
[status, latestValidHash, validationError] = engine_newPayloadV1(executionPayload)
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
- [`engine_newPayloadV1`] — executes an execution payload to create a block

[`engine_forkchoiceUpdatedV1`]: exec-engine.md#engine_forkchoiceUpdatedV1
[`engine_getPayloadV1`]: exec-engine.md#engine_newPayloadV1
[`engine_newPayloadV1`]: exec-engine.md#engine_newPayloadV1

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

All invocations of [`engine_forkchoiceUpdatedV1`], [`engine_getPayloadV1`] and [`engine_newPayloadV1`] by the
rollup driver should not result in errors assuming conformity with the specification. Said otherwise, all errors are
implementation concerns and it is up to them to handle them (e.g. by retrying, or by stopping the chain derivation and
requiring manual user intervention).

The following scenarios are assimilated to errors:

- [`engine_forkchoiceUpdatedV1`] returning a `status` of `"SYNCING"` instead of `"SUCCESS"` whenever passed a
  `headBlockHash` that it retrieved from a previous call to [`engine_newPayloadV1`].
- [`engine_newPayloadV1`] returning a `status` of `"SYNCING"` or `"INVALID"` whenever passed an execution payload
  that was obtained by a previous call to [`engine_getPayloadV1`].

### Finalization Guarantees

[finalization]: #finalization-guarantees

As stated earlier, an L2 block is considered *finalized* after a delay of `FINALIZATION_DELAY_BLOCKS == 50400` L1 blocks
after the L1 block that generated it. This is a duration of approximately 7 days worth of L1 blocks. This is also known
as the "fault proof window", as after this time the block can no longer be challenged by a fault proof.

L1 Ethereum reaches [finality][l1-finality] approximately every [12.8 minutes][consensus-time-params]. L2 blocks
generated from finalized L1 blocksare "safer" than most recent L2 blocks because they will never disappear from the
chain's history because of a re-org. However, they can still be challenged by a fault proof until the end of the fault
proof window.

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
a 32 byte hash corresponding to the [L2 output root](./proposals.md#l2-output-commitment-construction).

[SSZ]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md

### Output Method API

The input and return types here are as defined by the [engine API specs][engine-structures]).

[engine-structures]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#structures

- method: `optimism_outputAtBlock`
- params:
  1. `blockNumber`: `QUANTITY`, 64 bits - L2 integer block number </br>
        OR `String` - one of `"safe"`, `"latest"`, or `"pending"`.
- returns:
  1. `version`: `DATA`, 32 Bytes - the output root version number, beginning with 0.
  1. `l2OutputRoot`: `DATA`, 32 Bytes - the output root

# Handling L1 Re-Orgs

[l1-reorgs]: #handling-L1-re-orgs

The [previous section on L2 chain derivation][l2-chain-derivation] assumes linear progression of the L1 chain. It is
also applicable for batch processing, meaning that any given point in time, the canonical L2 chain is given by
processing the whole L1 chain since the [L2 chain inception][g-inception].

If the L1 Chain re-orgs, the rollup node must re-derive sections of the L2 chain such that it derives the same L2 chain
that a rollup node would derive if it only followed the new L1 chain.

> By itself, the previous section fully specifies the behavior of the rollup driver. **The current section is
> non-specificative** but shows how L1 re-orgs can be handled in practice.

In practice, the L1 chain is processed incrementally. However, the L1 chain may occasionally [re-organize][g-reorg],
meaning the head of the L1 chain changes to a block that is not the child of the previous head but rather another
descendant of an ancestor of the previous head. In that case, the rollup driver must first search for the common L1
ancestor, and can re-derive the L2 chain from that L1 block and onward.

The rollup node maintains two heads of the L2 Chain: the unsafe head (often called head) and the safe head.
Each L2 block has an L1 origin block (corresponding to its epoch) that it references in the
[L1 attributes deposited transaction][l1-attr-deposit]. The unsafe head is the head of the L2 chain.
Its L1 origin block should be canonical or potentially extending the canonical chain
(if the rollup node has not yet seen the L1 block that it is based upon).
The safe head is the the last L2 block of the last epoch whose sequencing window is complete
(i.e. the epoch with number `L1Head.number` - `SEQUENCING_WINDOW_SIZE`).

[l1-attr-deposit]: glossary.md#l1-attributes-deposited-transaction

Steps during a reorg:

1. Set "unsafe head" to equal the l2 head we retrieved, just as default
2. Set "latest block" to equal the l2 head we retrieved, also just as default
3. Walk back L2, and stop until block.l1Origin is found AND canonical, and update "latest block" to this block.
And don't override "unsafe head" if it's not found, but do override it when block.l1Origin does not match the
canonical L1 block at that height.
4. Walk back L2 from the "latest block" until a full sequencing window of L1 blocks has been passed.
This is the "safe block".

The purpose of this is to ensure that if the sequencing window for a L2 block has changed since it was derived,
that L2 block is re-derived.

The first L1 block of the sequencing window is the L1 attributes for that L2 block. The end of the sequencing
window is the canonical L1 block whose number is `SEQUENCING_WINDOW` larger than the start. The end of the
window must be selected by number otherwise the sequencer would not be able to create batches. The problem
with selecting the end of the window by number is that when an L1 reorg occurs, the blocks (and thus batches)
in the window could change. We must find the find the first L2 block whose complete sequencing window is
unchanged in the reorg.

When walking back on the L2 chain, care should be taken to not walk past the rollup genesis.

Note that post-[merge], the depth of re-orgs will be bounded by the [L1 finality delay][l1-finality] (every 2 epochs,
approximately 12 minutes).

(\*) Post-merge, this is only possible for 12 minutes. In practice, we'll pick an already-finalized L1 block as L2
inception point to preclude the possibility of a re-org past genesis, at the cost of a few empty blocks at the start of
the L2 chain.

[merge]: https://ethereum.org/en/eth2/merge/
