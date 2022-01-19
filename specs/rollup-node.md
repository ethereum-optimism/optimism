# Rollup Node Specification

<!-- All glossary references in this file. -->
[rollup node]: glossary.md#rollup-node
[derivation]: glossary.md#L2-chain-derivation
[payload attributes]: glossary.md#payload-attributes
[block]: glossary.md#block
[execution engine]: glossary.md#execution-engine
[reorg]: glossary.md#re-organization
[block gossip]: glossary.md#block-gossip
[rollup driver]: glossary.md#rollup-driver
[deposits]: glossary.md#deposits
[deposit feed contract]: glossary.md#L2-deposit-feed-contract
[L2 chain inception]: glossary.md#L2-chain-inception
[receipts]: glossary.md#receipt
[L1 attributes transaction]: glossary.md#l1-attributes-transaction
[transaction deposits]: glossary.md#transaction-deposits

The [rollup node] is the component responsible for [deriving the L2 chain][derivation] from L1 blocks (and their
associated [receipts]). This process happens in two steps:

1. Read from L1 blocks and associated receipts, in order to generate [payload attributes] (essentially [a block without
   output properties][block]).
2. Pass the payload attributes to the [execution engine], so that [output block properties][block] may be computed.

While this process is conceptually a pure function from the L1 chain to the L2 chain, it is in practice incremental. The
L2 chain is extended whenever new L1 blocks are added to the L1 chain. Similarly, the L2 chain re-organizes whenever the
L1 chain [re-organizes][reorg].

The part of the rollup node that derives the L2 chain is called the [rollup driver]. This document is currently only
concerned with the specification of the rollup driver.

## Table of Contents

- [L2 Chain Derivation](#l2-chain-derivation)
  - [Input derivation](#input-derivation)
    - [L1 attributes transaction derivation](#l1-attributes-transaction-derivation)
    - [Transaction deposits derivation](#transaction-deposits-derivation)
    - [Payload attributes derivation](#payload-attributes-derivation)
  - [Output derivation](#output-derivation)
- [Completing a driver step](#completing-a-driver-step)
  - [Execute](#execute)
  - [Forkchoice](#forkchoice)
- [API error handling](#api-error-handling)
- [Handling L1 Re-Orgs](#handling-l1-re-orgs)
- [Finalization Guarantees](#finalization-guarantees)

## L2 Chain Derivation

This section specifies how the [rollup driver] derives one L2 block per every L1 block.

First inputs are derived from L1 source data, then outputs are derived with L2 state through the [Engine API].

[Engine API]: exec-engine.md#engine-api

### Input derivation

The L2 block has the same format as a L1 block: a block-header and a list of transactions.

The list of transaction carries:

- A *[L1 attributes transaction]* (always first item)
- L2 transactions deposited by users in the L1 block (*[deposits]*, if any)

While deposits are notably (but not only) used to "deposit" (bridge) ETH and tokens to L2,
the word *deposit* should be understood as "a transaction *deposited* to L2".

The L1 attributes are read from the L1 block header, while other deposits are read from the block's [receipts].

All derived deposits each get two additional attributes during derivation, to ensure uniqueness:

- `blockHeight`: the block-height of the L1 input the deposit was derived from
- `transactionIndex`: the transaction-index within the L2 transactions list

#### L1 attributes transaction derivation

The rollup reads the following attributes from each L1 block to derive a [L1 attributes transaction]:

- block number
- timestamp
- basefee
- *random* (the output of the [`RANDOM` opcode][random])

These are then encoded as a [L1 attributes deposit] to update the [L1 Attributes Predeploy].

[random]: https://eips.ethereum.org/EIPS/eip-4399
[L1 attributes deposit]: deposits.md#l1-attributes-deposit
[L1 Attributes Predeploy]: deposits.md#l1-attributes-predeploy

#### Transaction deposits derivation

A [transaction deposit][transaction deposits] is an L2 transaction that has been submitted on L1, via a call to the
[deposit feed contract].

Refer to the[**deposit feed contract specification**][deposit-feed-spec] for details on how
deposit properties are emitted in deposit log entries.

[deposit-feed-spec]: deposits.md#deposit-feed-contract

#### Payload attributes derivation

From the data read from L1, the rollup node constructs an [expanded version of PayloadAttributesV1],
which includes an additional `transactions` field.

The object properties must be set as follows:

- `timestamp` is set to the current [unix time],
  rounded to the closest multiple of 2 seconds. No two blocks may have the same timestamp.
- `random` is set to the *random* L1 block attribute
- `suggestedFeeRecipient` is set to the zero-address for deposit-blocks, since there is no sequencer.
- `transactions` is an array of the derived deposits, all encoded in the [EIP-2718] format.

[expanded version of PayloadAttributesV1]: exec-engine.md#extended-payloadattributesv1
[unix time]: https://en.wikipedia.org/wiki/Unix_time
[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718
[EIP-2930]: https://eips.ethereum.org/EIPS/eip-2930

### Output derivation

Building a full block requires the earlier [derived payload attributes](#payload-attributes-derivation)
(`payloadAttributes` argument) as well as the previous L2 state (`forkchoiceState` argument), defined by
the [`engine_forkchoiceUpdatedV1`] method of the [Engine API]:

- `headBlockHash`: block hash of the last block of the L2 chain, according to the rollup driver.
- `safeBlockHash`: same as `headBlockHash`.
- `finalizedBlockHash`: the hash of the block whose number is `number(headBlockHash) - FINALIZATION_DELAY_BLOCKS` if
  the number of that block is `>= L2_CHAIN_INCEPTION`, 0 otherwise (where `FINALIZATION_DELAY_BLOCKS == 50400`
  (approximately 7 days worth of L1 blocks) and `L2_CHAIN_INCEPTION` is the [L2 chain inception] (the number of the
  first L1 block for which an L2 block was produced). See the [Finalization Guarantees][finalization] section for more
  details.

[`engine_forkchoiceUpdatedV1`]: exec-engine.md#engine_forkchoiceUpdatedV1

Once this first API call completes, `engine_getPayloadV1` is used to fetch the full L2 block,
as specified in the [Engine API].

[`engine_getPayloadV1`]: exec-engine.md#engine_executepayloadv1
[`ExecutionPayloadV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1

## Completing a driver step

After deriving a full L2 block, two more API calls are required to persist the result:
execute the new block, and update the forkchoice to reflect the new head.

### Execute

Execute through the [`engine_executePayloadV1`] API method with the derived payload to update the engine state.
A `"status": "VALID"` result is required to continue.

[`engine_executePayloadV1`]: exec-engine.md#engine_executepayloadv1

### Forkchoice

Update the L2 head with a [`engine_forkchoiceUpdatedV1`] API call, now without `payloadAttributes` argument,
and updated `forkchoiceState` argument:

- `headBlockHash`: block hash of the derived payload
- `safeBlockHash`: same as `headBlockHash`
- `finalizedBlockHash`: finalized-block. May have changed since last call.
   Not strictly required to change, this can be adjusted later.

A `"status": "SUCCESS"` result then indicates if the engine successfully updated the head to the derived payload.

## API error handling

All invocations of [`engine_forkchoiceUpdatedV1`], [`engine_getPayloadV1`] and [`engine_executePayloadV1`] by the
rollup driver should not result in errors assuming conformity with the specification. Said otherwise, all errors are
implementation concerns and it is up to them to handle them (e.g. by retrying, or by stopping the chain derivation and
requiring manual user intervention).

The following scenarios are assimilated to errors:

- [`engine_forkchoiceUpdatedV1`] returning a `status` of `"SYNCING"` instead of `"SUCCESS"` whenever passed a
  `headBlockHash` that it retrieved from a previous call to [`engine_executePayloadV1`].
- [`engine_executePayloadV1`] returning a `status` of `"SYNCING"` or `"INVALID"` whenever passed an execution payload
  that was obtained by a previous call to [`engine_getPayloadV1`].

## Handling L1 Re-Orgs

[l1-reorgs]: #handling-L1-re-orgs

The [previous section on L2 chain derivation][l2-chain-derivation] assumes linear progression of the L1 chain. It is
also applicable for batch processing, meaning that any given point in time, the canonical L2 chain is given by
processing the whole L1 chain since the [L2 chain inception].

> By itself, the previous section fully specifies the behaviour of the rollup driver. **The current section is
> non-specificative** but shows how L1 re-orgs can be handled in practice.

In practice, the L1 chain is processed incrementally. However, the L1 chain may occasionally re-organize, meaning the
head of the L1 chain changes to a block that is not the child of the previous head but rather some other descendant
of an ancestor of the previous head. In that case, the rollup driver must first search for common ancestor, and can then
continue deriving with the new canonical L1 block after the common point.

This sync starting point (L1 block to derive from, and L2 parent to build on) is determined by:

- Retrieve the head block of the engine (`refL2`), then determine the L1 block it was derived from (`refL1`),
  and where it builds on (`parentL2`).
- Fetch the L1 block at the same height (`currentL1`):
  - If not found: consider this a reorg to a shorter L1 chain, continue.
- If the L1 source considers this canonical (`currentL1 == refL1`):
  - Find the next L1 block (it may not exist yet) and return that as `nextRefL1`, along with the `refL2`.
    - Note: after looking up `N+1` ensure L1 has not changed during block-by-number lookups (`refL1 == nextL1_parent`).
- While have not found a block in the engine common with the canonical chain, traverse the L2 chain back until genesis:
  - Each step starts by caching the previous `currentL1` as `nextRefL1`.
  - Lookup the parent by hash: Each step `refL2` should equal the previous `parentL2`.
    `refL1` and `parentL2` are also traversed back by parsing the `refL2` block.
  - The canonical L1 block is looked up at the same height (`currentL1`).
    - If not found: consider this a reorg to a shorter L1 chain, continue.
  - If the engine and canonical chain match (`refL1 == currentL1`), then return that as `nextRefL1`, along with `refL2`.
- If there are no common blocks after genesis, check if `refL2` and `currentL1` match the expected genesis blocks.
- If the genesis is correct, the last cached `nextRefL1` is returned, along with the L2 genesis.

> Note that post-[merge], the L1 chain will offer finalization guarantees meaning that it won't be able to re-org more
> than `FINALIZATION_DELAY_BLOCKS == 50400` in the past, hence preserving our finalization guarantees.

[merge]: https://ethereum.org/en/eth2/merge/

Just like before, the meaning of errors returned by RPC calls is unspecified and must be handled at the implementer's
discretion, while remaining compatible with the specification.

## Finalization Guarantees

[finalization]: #finalization-guarantees

As already alluded to in the section on [interacting with the execution engine][calling-exec-engine], an L2 block is
considered *finalized* after a delay of `FINALIZATION_DELAY_BLOCKS == 50400` blocks after the L1 block that generated
it. This is a duration of approximately 7 days worth of L1 blocks.

L1 Ethereum [reaches finality approximately every 12 minutes][l1-finality], so these L2 blocks can safely be considered
to be final: they will never disappear from the chain's history because of a re-org.

[l1-finality]: https://www.paradigm.xyz/2021/07/ethereum-reorgs-after-the-merge/
