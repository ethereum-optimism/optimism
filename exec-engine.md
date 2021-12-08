# L2 Execution Engine

This document outlines the modifications, configuration and usage of a L1 execution engine for L2.

## Deposit processing

The Engine interfaces abstract away transaction types with [EIP-2718][eip-2718].

To support rollup functionality, processing of a new Deposit [`TransactionType`][eip-2718-transactions]
is implemented by the engine, see the [deposit specification][deposit-spec].

This type of transaction can mint L2 ETH, run EVM,
and introduce L1 information to enshrined contracts in the execution state.

> **TODO**: implement deposit spec doc

[deposit-spec]: ./deposits#deposit-transaction-type

### Deposit boundaries

Transactions cannot be blindly trusted, trust is established through authentication.
Unlike other transaction types deposits are not authenticated by a signature:
the rollup node authenticates them, outside of the engine.

To process deposits safely, the deposits MUST be authenticated first:

- Ingest directly through trusted Engine API
- Part of sync towards a trusted block hash (trusted through previous Engine API instruction)

Deposits MUST never be consumed from the transaction pool.
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
to [`engine_forkchoiceUpdatedV1`][engine_forkchoiceUpdatedV1]:

[`PayloadAttributesV1`][PayloadAttributesV1] is extended with a `transactions` field,
equivalent to the `transactions` field in [`ExecutionPayloadV1`][ExecutionPayloadV1]:
> `Array of DATA` - Array of transaction objects, each object is a byte list ([`DATA`][exec-api-data]) representing
> `TransactionType || TransactionPayload` or `LegacyTransaction` as defined in [EIP-2718][eip-2718].

This `transactions` field is an optional JSON field:

- If empty or missing: no changes to engine behavior.
  Utilized by sequencers (if enabled) to consume the transaction pool.
- If present and non-empty: the payload MUST only be produced with this exact list of transactions.
  Utilized by [rollup driver][rollup-driver] to compute full block payloads based on deterministic inputs.
  
> **TODO**: derivation function spec in rollup node doc or separate driver doc

[rollup-driver]: ./rollup-driver.md

### `engine_executePayloadV1`

No modifications to [`engine_executePayloadV1`][engine_executePayloadV1].
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

1. Engine API informs engine of chain head, unconditionally (part of regular node operation):
   - [`engine_executePayloadV1`][engine_executePayloadV1] is called with latest L2 block derived from L1.
   - [`engine_forkchoiceUpdatedV1`][engine_forkchoiceUpdatedV1] is called with the current
     `unsafe`/`safe`/`finalized` L2 block hashes.
2. Engine requests headers from peers, in reverse till the parent hash matches the local chain
3. Engine catches up:
    a) A form of state sync is activated towards the finalized or head block hash
    b) A form of block sync pulls block bodies and processes towards head block hash

The exact P2P based sync is out of scope for the L2 specification:
the operation within the engine is the exact same as with L1 (although with an EVM that supports deposits).

### Worst-case sync

1. Engine is out of sync, not peered and/or stalled due other reasons.
2. rollup node periodically fetches latest head from engine (`eth_getBlockByNumber`)
3. rollup node activates sync if the engine is out of sync but not syncing through P2P (`eth_syncing`)
4. rollup node inserts blocks, derived from L1, one by one,
   starting from the engine head (or genesis block if unrecognized) up to the latest chain head.
   (`engine_forkchoiceUpdatedV1`, `engine_executePayloadV1`)

See [rollup node sync spec][rollup-node-sync] for L1-based block syncing specification.

> **TODO**: rollup node block-by-block sync (covered in rollup node PR #43)

[rollup-node-sync]: ./rollup-node.md#sync

[eip-2718]: https://eips.ethereum.org/EIPS/eip-2718
[eip-2718-transactions]: https://eips.ethereum.org/EIPS/eip-2718#transactions
[exec-api-data]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#structures
[l1-api-spec]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md
[PayloadAttributesV1]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#PayloadAttributesV1
[ExecutionPayloadV1]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#ExecutionPayloadV1
[engine_forkchoiceUpdatedV1]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_forkchoiceupdatedv1
[engine_executePayloadV1]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_executePayloadV1
[engine_getPayloadV1]: https://github.com/ethereum/execution-apis/blob/v1.0.0-alpha.5/src/engine/specification.md#engine_getPayloadV1
