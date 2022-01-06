# Rollup Node Specification

<!-- All glossary references in this file. -->
[rollup node]: /glossary.md#rollup-node
[derivation]: /glossary.md#L2-chain-derivation
[payload attributes]: /glossary.md#payload-attributes
[block]: /glossary.md#block
[execution engine]: /glossary.md#execution-engine
[reorg]: /glossary.md#re-organization
[block gossip]: /glossary.md#block-gossip
[rollup driver]: /glossary.md#rollup-driver
[deposits]: /glossary.md#deposits
[deposit-feed]: /glossary.md#L2-deposit-feed-contract
[L2 chain inception]: /glossary.md#L2-chain-inception
[receipts]: /glossary.md#receipt

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

- [L2 Chain Derivation][l2-chain-derivation]
  - [From L1 blocks to payload attributes][payload-attr]
  - [Payload Transaction Format][payload-format]
  - [Building the L2 block with the execution engine][calling-exec-engine]
- [Handling L1 Re-Orgs][l1-reorgs]
- [Finalization Guarantees][finalization]

## L2 Chain Derivation

[l2-chain-derivation]: #l2-chain-derivation

This section specifies how the [rollup driver] derives one L2 block per every L1 block. The L2 block will carry the L1
block attributes (as a *[L1 attributes transaction]*) well as all L2 transactions deposited by users in the L1 block
(*[deposits]*).

[L1 attributes transaction]: /glossary.md#l1-attributes-transaction

### From L1 blocks to payload attributes

[payload-attr]: #From-L1-blocks-to-payload-attributes
[`PayloadAttributesOPV1`]: #From-L1-blocks-to-payload-attributes

The rollup reads the following data from each L1 block:

- L1 block attributes
  - block number
  - timestamp
  - basefee
  - *random* (the output of the [`RANDOM` opcode][random])
- [deposits]

[random]: https://eips.ethereum.org/EIPS/eip-4399

A deposit is an L2 transaction that has been submitted on L1, via a call to the [deposit feed contract][deposit-feed].

While deposits are notably (but not only) used to "deposit" (bridge) ETH and tokens to L2, the word *deposit* should be
understood as "a transaction *deposited* to L2".

The L1 attributes are read from the L1 block header, while deposits are read from the block's [receipts]. Refer to the
[**deposit feed contract specification**][deposit-feed-spec] for details on how deposits are encoded as log entries.

[deposit-feed-spec]: deposits.md#deposit-feed-contract

From the data read from L1, the rollup node constructs an expanded version of the [Engine API PayloadAttributesV1
object][PayloadAttributesV1], which includes an additional `transactions` field:

[PayloadAttributesV1]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#payloadattributesv1

```js
PayloadAttributesOPV1: {
    timestamp: QUANTITY
    random: DATA (32 bytes)
    suggestedFeeRecipient: DATA (20 bytes)
    transactions: array of DATA
}
```

The type notation used here refers to the [HEX value encoding] used by the [Ethereum JSON-RPC API
specification][JSON-RPC-API], as this structure will need to be sent over JSON-RPC. `array` refers to a JSON array.

[HEX value encoding]: https://eth.wiki/json-rpc/API#hex-value-encoding
[JSON-RPC-API]: https://github.com/ethereum/execution-apis

The object properties must be set as follows:

- `timestamp` is set to the current [unix time] (number of elapsed seconds since 00:00:00 UTC on 1 January 1970),
  rounded to the closest multiple of 2 seconds. No two blocks may have the same timestamp.
- `random` is set to the *random* L1 block attribute
- `suggestedFeeRecipient` is set to an address where the sequencer would like to direct the fees
- `transactions` is an array of transactions, encoded in the [EIP-2718] format (i.e. as a single byte defining the
  transaction type, concatenated with an opaque byte array whose meaning depends on the type).

> **TODO** we need to handle non-EIP-2718 transactions too

[unix type]: https://en.wikipedia.org/wiki/Unix_time
[merge]: https://ethereum.org/en/eth2/merge/
[EIP-2718]: https://eips.ethereum.org/EIPS/eip-2718

[encode-tx]: https://github.com/norswap/nanoeth/blob/cc5d94a349c90627024f3cd629a2d830008fec72/src/com/norswap/nanoeth/transactions/Transaction.java#L84-L130

The [EIP-2718] transactions must have a transaction type that is valid on L1, or be an *[L1 attributes transaction]*
(see below).

#### Payload Transaction Format

[payload-format]: #payload-transaction-format

The `transactions` array is filled with the deposits, prefixed by the (single) [L1 attributes transaction]. The deposits
are simply copied byte-for-byte — it is the role of the [execution engine] to reject invalidly-formatted transactions.

> **TODO** must offer some precisions on the format of deposits: sender,
> receivers both in-tx-as-encoded, and on-L2-tx. What about the fees?

The Optimism-specific *[L1 attributes transaction]* has the following [EIP-2718]-compatible format: `0x7E ||
[block_number, timestamp, basefee]` where:

- `0x7E` is the transaction type identifier.
- `block_number` is the L1 block number as a 64-bit integer (4 bytes)
- `timestamp` is the L1 block timestamp as a 64-bit integer (4 bytes)
- `basefee` is the L1 block basefee as a 64-bit integer (4 bytes)

When included in the `transactions` array, this transaction should be RLP-encoded in the same way as other transactions.

> **TODO** move this section into a doc specific to the execution-engine

Here is an example valid `PayloadAttributesOPV1` object, which contains an L1 attributes transaction as well as a single
deposit:

```js
{
  timestamp: "0x61a6336f",
  random: "0xde5dff2b0982ecbbd38081eb8f4aed0525140dc1c1d56f995b4fa801a3f2649e",
  suggestedFeeRecipient: "0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B",
  transactions: [
    "TODO specify L1 attribute transaction",
    "0x02f87101058459682f0085199c82cc0082520894ab5801a7d398351b8be11c439e05c5b3259aec9b8609184e72a00080c080a0a6d217a91ea344fc09f740f104f764d71bb1ca9a8e159117d2d27091ea5fce91a04cf5add5f5b7d791a2c4663ab488cb581df800fe0910aa755099ba466b49fd69"
  ]
}
```

### Building the L2 block with the execution engine

[calling-exec-engine]: #building-the-L2-block-with-the-execution-engine

The Optimism [execution engine] is specified in the [Execution Engine Specification].

[Execution Engine Specification]: exec-engine.md

This section defines how the rollup driver must interact with the execution engine's in order to convert [payload
attributes] into L2 blocks.

> **TODO** This section probably includes too much redundant details that will
> need to be removed once the execution engine spec is up.

Optimism's execution engine API is built upon [Ethereum's Engine API specification][eth-engine-api], with a
couple of modifications. That specification builds upon [Ethereum's JSON-RPC API specification][JSON-RPC-API], which
itself builds upon the [JSON-RPC specification][JSON-RPC].

[eth-engine-api]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md
[JSON-RPC]: https://www.jsonrpc.org/specification

In particular, the [Ethereum's Engine API specification][eth-engine-api] specifies a [JSON-RPC] endpoint with a number
of JSON-RPC routes, which are the means through which the rollup driver interacts with the execution engine.

Instead of calling [`engine_forkchoiceUpdatedV1`], the rollup driver must call the new [`engine_forkchoiceUpdatedOPV1`]
route. This has the same signature, except that:

[`engine_forkchoiceUpdatedV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#engine_forkchoiceupdatedv1
[`engine_forkchoiceUpdatedOPV1`]: exec-engine.md#engine_forkchoiceupdatedv1

- it takes a [`PayloadAttributesOPV1`] object as input instead of [`PayloadAttributesV1`][PayloadAttributesV1]. The
  execution engine must include the valid transactions supplied in this object in the block, in the same order as they
  were supplied, and only those. See the [previous section][payload-attr] for the specification of how the properties
  must be set.

- we repurpose the [`ForkchoiceStateV1`] structure with the following property semantics:
  - `headBlockHash`: block hash of the last block of the L2 chain, according to the rollup driver.
  - `safeBlockHash`: same as `headBlockHash`.
  - `finalizedBlockHash`: the hash of the block whose number is `number(headBlockHash) - FINALIZATION_DELAY_BLOCKS` if
    the number of that block is `>= L2_CHAIN_INCEPTION`, 0 otherwise (where `FINALIZATION_DELAY_BLOCKS == 50400`
    (approximately 7 days worth of L1 blocks) and `L2_CHAIN_INCEPTION` is the [L2 chain inception] (the number of the
    first L1 block for which an L2 block was produced). See the [Finalization Guarantees][finalization] section for more
    details.

[`ForkchoiceStateV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#ForkchoiceStateV1

> **Note:** the properties of this `ForkchoiceStateV1` can be used to anchor queries to the regular (non-engine-API)
> JSON-RPC endpoint of the execution engine. [See here for more information.][L2-JSON-RPC-API]

[L2-JSON-RPC-API]: TODO

> **TODO LINK** L2 JSON RPC API (might be the same as [L1's][JSON-RPC-API])

The `payloadID` returned by [`engine_forkchoiceUpdatedOPV1`] can then be passed to [`engine_getPayloadV1`] in order to
obtain an [`ExecutionPayloadV1`], which fully defines a new L2 block.

The rollup driver must then instruct the execution engine to execute the block by calling [`engine_executePayloadV1`].
This returns the new L2 block hash.

All invocations of [`engine_forkchoiceUpdatedOPV1`], [`engine_getPayloadV1`] and [`engine_executePayloadV1`] by the
rollup driver should not result in errors assuming conformity with the specification. Said otherwise, all errors are
implementation concerns and it is up to them to handle them (e.g. by retrying, or by stopping the chain derivation and
requiring manual user intervention).

The following scenarios are assimilated to errors:

- [`engine_forkchoiceUpdateOPV1`] returning a `status` of "`SYNCING`" instead of "`SUCCESS`" whenever passed a
  `headBlockHash` that it retrieved from a previous call to [`engine_executePayloadV1`].
- [`engine_executePayloadV1`] returning a `status` of "`SYNCING`" or `"INVALID"` whenever passed an execution payload
  that was obtained by a previous call to [`engine_getPayloadV1`].

[`engine_getPayloadV1`]: exec-engine.md#engine_executepayloadv1
[`ExecutionPayloadV1`]: https://github.com/ethereum/execution-apis/blob/main/src/engine/specification.md#executionpayloadv1
[`engine_executePayloadV1`]: exec-engine.md#engine_executepayloadv1

## Handling L1 Re-Orgs

[l1-reorgs]: #handling-L1-re-orgs

The [previous section on L2 chain derivation][l2-chain-derivation] assumes linear progression of the L1 chain. It is
also applicable for batch processing, meaning that any given point in time, the canonical L2 chain is given by
processing the whole L1 chain since the [L2 chain inception].

> By itself, the previous section fully specifies the behaviour of the rollup driver. **The current section is
> non-specificative** but shows how L1 re-orgs can be handled in practice.

In practice, the L1 chain is processed incrementally. However, the L1 chain may occasionally re-organize, meaning the
head of the L1 chain changes to a block that is not the child of the previous head but rather one of its "cousins" (i.e.
the descendant of an ancestor of the previous head). In those case, the rollup driver must:

1. Call [`engine_forkchoiceUpdatedOPV1`] for the new L2 chain head
    - Pass `null` for the `payloadAttributes` parameter.
    - Fill the [`ForkchoiceStateV1`] object according to [the section on the execution engine][calling-exec-engine], but
      set `headBlockHash` to the hash of the new L2 chain head. `safeBlockHash` and `finalizedBlockHash` must be updated
      accordingly.
2. If the call returns `"SUCCESS"`, we are done: the execution engine retrieved all the new L2 blocks via [block sync].
3. Otherwise the call returns `"SYNCING"`, and we must derive the new blocks ourselves. Start by locating the *common
   ancestor*, a block that is an ancestor of both the previous and new head.
4. Isolate the range of L1 blocks from `common ancestor` (excluded) to `new head` (included).
5. For each such block, call [`engine_forkchoiceUpdatedOPV1`], [`engine_getPayloadV1`], and [`engine_executePayloadV1`].
   - Fill the [`PayloadAttributesOPV1`] object according to [the section on payload attributes][payload-attr].
   - Fill the [`ForkchoiceStateV1`] object according to [the section on the execution engine][calling-exec-engine], but
     set `headBlockHash` to the hash of the last processed L2 block (use the hash of the common ancestor initially)
     instead of the last L2 chain head. `safeBlockHash` and `finalizedBlockHash` must be updated accordingly.

[block sync]: https://github.com/ethereum-optimism/optimistic-specs/blob/main/exec-engine.md#sync

> Note that post-[merge], the L1 chain will offer finalization guarantees meaning that it won't be able to re-org more
> than `FINALIZATION_DELAY_BLOCKS == 50400` in the past, hence preserving our finalization guarantees.

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
