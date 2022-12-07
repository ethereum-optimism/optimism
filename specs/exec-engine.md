# L2 Execution Engine

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Deposited transaction processing](#deposited-transaction-processing)
  - [Deposited transaction boundaries](#deposited-transaction-boundaries)
- [Engine API](#engine-api)
  - [`engine_forkchoiceUpdatedV1`](#engine_forkchoiceupdatedv1)
    - [Extended PayloadAttributesV1](#extended-payloadattributesv1)
  - [`engine_newPayloadV1`](#engine_newpayloadv1)
  - [`engine_getPayloadV1`](#engine_getpayloadv1)
- [Networking](#networking)
- [Sync](#sync)
  - [Happy-path sync](#happy-path-sync)
  - [Worst-case sync](#worst-case-sync)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document outlines the modifications, configuration and usage of a L1 execution engine for L2.

## Deposited transaction processing

The Engine interfaces abstract away transaction types with [EIP-2718][eip-2718].

To support rollup functionality, processing of a new Deposit [`TransactionType`][eip-2718-transactions]
is implemented by the engine, see the [deposits specification][deposit-spec].

This type of transaction can mint L2 ETH, run EVM,
and introduce L1 information to enshrined contracts in the execution state.

[deposit-spec]: deposits.md

### Deposited transaction boundaries

Transactions cannot be blindly trusted, trust is established through authentication.
Unlike other transaction types deposits are not authenticated by a signature:
the rollup node authenticates them, outside of the engine.

To process deposited transactions safely, the deposits MUST be authenticated first:

- Ingest directly through trusted Engine API
- Part of sync towards a trusted block hash (trusted through previous Engine API instruction)

Deposited transactions MUST never be consumed from the transaction pool.
*The transaction pool can be disabled in a deposits-only rollup*

## Engine API

*Note: the [Engine API][l1-api-spec] is in alpha, `v1.0.0-alpha.5`.
There may be subtle tweaks, beta starts in a few weeks*

### `engine_forkchoiceUpdatedV1`

This updates which L2 blocks the engine considers to be canonical (`forkchoiceState` argument),
and optionally initiates block production (`payloadAttributes` argument).

Within the rollup, the types of forkchoice updates translate as:

- `headBlockHash`: block hash of the head of the canonical chain. Labeled `"unsafe"` in user JSON-RPC.
   Nodes may apply L2 blocks out of band ahead of time, and then reorg when L1 data conflicts.
- `safeBlockHash`: block hash of the canonical chain, derived from L1 data, unlikely to reorg.
- `finalizedBlockHash`: irreversible block hash, matches lower boundary of the dispute period.

To support rollup functionality, one backwards-compatible change is introduced
to [`engine_forkchoiceUpdatedV1`][engine_forkchoiceUpdatedV1]: the extended `PayloadAttributesV1`

#### Extended PayloadAttributesV1

[`PayloadAttributesV1`][PayloadAttributesV1] is extended to:

```js
PayloadAttributesV1: {
    timestamp: QUANTITY
    random: DATA (32 bytes)
    suggestedFeeRecipient: DATA (20 bytes)
    transactions: array of DATA
    noTxPool: bool
    gasLimit: QUANTITY or null
}
```

The type notation used here refers to the [HEX value encoding] used by the [Ethereum JSON-RPC API
specification][JSON-RPC-API], as this structure will need to be sent over JSON-RPC. `array` refers
to a JSON array.

Each item of the `transactions` array is a byte list encoding a transaction: `TransactionType ||
TransactionPayload` or `LegacyTransaction`, as defined in [EIP-2718][eip-2718].
This is equivalent to the `transactions` field in [`ExecutionPayloadV1`][ExecutionPayloadV1]

The `transactions` field is optional:

- If empty or missing: no changes to engine behavior. The sequencers will (if enabled) build a block
  by consuming transactions from the transaction pool.
- If present and non-empty: the payload MUST be produced starting with this exact list of transactions.
  The [rollup driver][rollup-driver] determines the transaction list based on deterministic L1 inputs.

The `noTxPool` is optional as well, and extends the `transactions` meaning:

- If `false`, the execution engine is free to pack additional transactions from external sources like the tx pool
  into the payload, after any of the `transactions`. This is the default behavior a L1 node implements.
- If `true`, the execution engine must not change anything about the given list of `transactions`.

If the `transactions` field is present, the engine must execute the transactions in order and return `STATUS_INVALID`
if there is an error processing the transactions. It must return `STATUS_VALID` if all of the transactions could
be executed without error. **Note**: The state transition rules have been modified such that deposits will never fail
so if `engine_forkchoiceUpdatedV1` returns `STATUS_INVALID` it is because a batched transaction is invalid.

The `gasLimit` is optional w.r.t. compatibility with L1, but required when used as rollup.
This field overrides the gas limit used during block-building.
If not specified as rollup, a `STATUS_INVALID` is returned.

[rollup-driver]: rollup-node.md

### `engine_newPayloadV1`

No modifications to [`engine_newPayloadV1`][engine_newPayloadV1].
Applies a L2 block to the engine state.

### `engine_getPayloadV1`

No modifications to [`engine_getPayloadV1`][engine_getPayloadV1].
Retrieves a payload by ID, prepared by `engine_forkchoiceUpdatedV1` when called with `payloadAttributes`.

## Networking

The execution engine can acquire all data through the rollup node, as derived from L1:
*P2P networking is strictly optional.*

However, to not bottleneck on L1 data retrieval speed, the P2P network functionality SHOULD be enabled, serving:

- Peer discovery ([Disc v5][discv5])
- [`eth/66`][eth66]:
  - Transaction pool (consumed by sequencer nodes)
  - State sync (happy-path for fast trustless db replication)
  - Historical block header and body retrieval
  - *New blocks are acquired through the consensus layer instead (rollup node)*

No modifications to L1 network functionality are required, except configuration:

- [`networkID`][network-id]: Distinguishes the L2 network from L1 and testnets.
  Equal to the [`chainID`][chain-id] of the rollup network.
- Activate Merge fork: Enables Engine API and disables propagation of blocks,
  as block headers cannot be authenticated without consensus layer.
- Bootnode list: DiscV5 is a shared network,
  [bootstrap][discv5-rationale] is faster through connecting with L2 nodes first.

[discv5]: https://github.com/ethereum/devp2p/blob/master/discv5/discv5.md
[eth66]: https://github.com/ethereum/devp2p/blob/master/caps/eth.md
[network-id]: https://github.com/ethereum/devp2p/blob/master/caps/eth.md#status-0x00
[chain-id]: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md
[discv5-rationale]: https://github.com/ethereum/devp2p/blob/master/discv5/discv5-rationale.md

## Sync

The execution engine can operate sync in different ways:

- Happy-path: rollup node informs engine of the desired chain head as determined by L1, completes through engine P2P.
- Worst-case: rollup node detects stalled engine, completes sync purely from L1 data, no peers required.

The happy-path is more suitable to bring new nodes online quickly,
as the engine implementation can sync state faster through methods like [snap-sync][snap-sync].

[snap-sync]: https://github.com/ethereum/devp2p/blob/master/caps/snap.md

### Happy-path sync

1. The rollup node informs the engine of the L2 chain head, unconditionally (part of regular node operation):
   - [`engine_newPayloadV1`][engine_newPayloadV1] is called with latest L2 block derived from L1.
   - [`engine_forkchoiceUpdatedV1`][engine_forkchoiceUpdatedV1] is called with the current
     `unsafe`/`safe`/`finalized` L2 block hashes.
2. The engine requests headers from peers, in reverse till the parent hash matches the local chain
3. The engine catches up:
    a) A form of state sync is activated towards the finalized or head block hash
    b) A form of block sync pulls block bodies and processes towards head block hash

The exact P2P based sync is out of scope for the L2 specification:
the operation within the engine is the exact same as with L1 (although with an EVM that supports deposits).

### Worst-case sync

1. Engine is out of sync, not peered and/or stalled due other reasons.
2. The rollup node maintains latest head from engine (poll `eth_getBlockByNumber` and/or maintain a header subscription)
3. The rollup node activates sync if the engine is out of sync but not syncing through P2P (`eth_syncing`)
4. The rollup node inserts blocks, derived from L1, one by one, potentially adapting to L1 reorg(s),
   as outlined in the [rollup node spec] (`engine_forkchoiceUpdatedV1`, `engine_newPayloadV1`)

[rollup node spec]: rollup-node.md

[eip-2718]: https://eips.ethereum.org/EIPS/eip-2718
[eip-2718-transactions]: https://eips.ethereum.org/EIPS/eip-2718#transactions
[exec-api-data]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#structures
[l1-api-spec]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md
[PayloadAttributesV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#PayloadAttributesV1
[ExecutionPayloadV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#ExecutionPayloadV1
[engine_forkchoiceUpdatedV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#engine_forkchoiceupdatedv1
[engine_newPayloadV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#engine_newPayloadV1
[engine_getPayloadV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#engine_getPayloadV1
[HEX value encoding]: https://eth.wiki/json-rpc/API#hex-value-encoding
[JSON-RPC-API]: https://github.com/ethereum/execution-apis
