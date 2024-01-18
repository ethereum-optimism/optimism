# L2 Execution Engine

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [1559 Parameters](#1559-parameters)
- [Deposited transaction processing](#deposited-transaction-processing)
  - [Deposited transaction boundaries](#deposited-transaction-boundaries)
- [Fees](#fees)
  - [Fee Vaults](#fee-vaults)
  - [Priority fees (Sequencer Fee Vault)](#priority-fees-sequencer-fee-vault)
  - [Base fees (Base Fee Vault)](#base-fees-base-fee-vault)
  - [L1-Cost fees (L1 Fee Vault)](#l1-cost-fees-l1-fee-vault)
    - [Pre-Ecotone](#pre-ecotone)
    - [Ecotone L1-Cost fee changes (EIP-4844 DA)](#ecotone-l1-cost-fee-changes-eip-4844-da)
- [Engine API](#engine-api)
  - [`engine_forkchoiceUpdatedV2`](#engine_forkchoiceupdatedv2)
    - [Extended PayloadAttributesV2](#extended-payloadattributesv2)
  - [`engine_forkchoiceUpdatedV3`](#engine_forkchoiceupdatedv3)
    - [Extended PayloadAttributesV3](#extended-payloadattributesv3)
  - [`engine_newPayloadV2`](#engine_newpayloadv2)
  - [`engine_newPayloadV3`](#engine_newpayloadv3)
  - [`engine_getPayloadV2`](#engine_getpayloadv2)
  - [`engine_getPayloadV3`](#engine_getpayloadv3)
    - [Extended Response](#extended-response)
  - [`engine_signalSuperchainV1`](#engine_signalsuperchainv1)
- [Networking](#networking)
- [Sync](#sync)
  - [Happy-path sync](#happy-path-sync)
  - [Worst-case sync](#worst-case-sync)
- [Ecotone: disable Blob-transactions](#ecotone-disable-blob-transactions)
- [Ecotone: Beacon Block Root](#ecotone-beacon-block-root)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

This document outlines the modifications, configuration and usage of a L1 execution engine for L2.

## 1559 Parameters

The execution engine must be able to take a per chain configuration which specifies the EIP-1559 Denominator
and EIP-1559 elasticity. After Canyon it should also take a new value `EIP1559DenominatorCanyon` and use that as
the denominator in the 1559 formula rather than the prior denominator.

The formula for EIP-1559 is not otherwise modified.

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

## Fees

Sequenced transactions (i.e. not applicable to deposits) are charged with 3 types of fees:
priority fees, base fees, and L1-cost fees.

### Fee Vaults

The three types of fees are collected in 3 distinct L2 fee-vault deployments for accounting purposes:
fee payments are not registered as internal EVM calls, and thus distinguished better this way.

These are hardcoded addresses, pointing at pre-deployed proxy contracts.
The proxies are backed by vault contract deployments, based on `FeeVault`, to route vault funds to L1 securely.

| Vault Name          | Predeploy                                                |
|---------------------|----------------------------------------------------------|
| Sequencer Fee Vault | [`SequencerFeeVault`](./predeploys.md#SequencerFeeVault) |
| Base Fee Vault      | [`BaseFeeVault`](./predeploys.md#BaseFeeVault)           |
| L1 Fee Vault        | [`L1FeeVault`](./predeploys.md#L1FeeVault)               |

### Priority fees (Sequencer Fee Vault)

Priority fees follow the [eip-1559] specification, and are collected by the fee-recipient of the L2 block.
The block fee-recipient (a.k.a. coinbase address) is set to the Sequencer Fee Vault address.

### Base fees (Base Fee Vault)

Base fees largely follow the [eip-1559] specification, with the exception that base fees are not burned,
but add up to the Base Fee Vault ETH account balance.

### L1-Cost fees (L1 Fee Vault)

The protocol funds batch-submission of sequenced L2 transactions by charging L2 users an additional fee
based on the estimated batch-submission costs.
This fee is charged from the L2 transaction-sender ETH balance, and collected into the L1 Fee Vault.

The exact L1 cost function to determine the L1-cost fee component of a L2 transaction depends on
the upgrades that are active.

#### Pre-Ecotone

Before Ecotone activation, L1 cost is calculated as:
`(rollupDataGas + l1FeeOverhead) * l1BaseFee * l1FeeScalar / 1e6` (big-int computation, result
in Wei and `uint256` range)
Where:

- `rollupDataGas` is determined from the *full* encoded transaction
  (standard EIP-2718 transaction encoding, including signature fields):
  - Before Regolith fork: `rollupDataGas = zeroes * 4 + (ones + 68) * 16`
    - The addition of `68` non-zero bytes is a remnant of a pre-Bedrock L1-cost accounting function,
       which accounted for the worst-case non-zero bytes addition to complement unsigned transactions, unlike Bedrock.
  - With Regolith fork: `rollupDataGas = zeroes * 4 + ones * 16`
- `l1FeeOverhead` is the Gas Price Oracle `overhead` value.
- `l1FeeScalar` is the Gas Price Oracle `scalar` value.
- `l1BaseFee` is the L1 base fee of the latest L1 origin registered in the L2 chain.

Note that the `rollupDataGas` uses the same byte cost accounting as defined in [eip-2028],
except the full L2 transaction now counts towards the bytes charged in the L1 calldata.
This behavior matches pre-Bedrock L1-cost estimation of L2 transactions.

Compression, batching, and intrinsic gas costs of the batch transactions are accounted for by the protocol
with the Gas Price Oracle `overhead` and `scalar` parameters.

The Gas Price Oracle `l1FeeOverhead` and `l1FeeScalar`, as well as the `l1BaseFee` of the L1 origin,
can be accessed in two interchangeable ways:

- read from the deposited L1 attributes (`l1FeeOverhead`, `l1FeeScalar`, `basefee`) of the current L2 block
- read from the L1 Block Info contract (`0x4200000000000000000000000000000000000015`)
  - using the respective solidity `uint256`-getter functions (`l1FeeOverhead`, `l1FeeScalar`, `basefee`)
  - using direct storage-reads:
    - L1 basefee as big-endian `uint256` in slot `1`
    - Overhead as big-endian `uint256` in slot `5`
    - Scalar as big-endian `uint256` in slot `6`

#### Ecotone L1-Cost fee changes (EIP-4844 DA)

Ecotone allows posting batches via Blobs which are subject to a new fee market. To account for this feature,
L1 cost is computed as:

`(zeroes*4 + ones*16) * (16*l1BaseFee*l1BaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar) / 16e6`

Where:

- the computation is an unlimited precision integer computation, with the result in Wei and having
  `uint256` range.

- zeoroes and ones are the count of zero and non-zero bytes respectively in the *full* encoded
  signed transaction.

- `l1BaseFee` is the L1 base fee of the latest L1 origin registered in the L2 chain.

- `l1BlobBaseFee` is the blob gas price, computed as described in [EIP-4844][4844-gas] from the
  header of the latest registered L1 origin block.

Conceptually what the above function captures is the formula below, where `compressedTxSize =
(zeroes*4 + ones*16) / 16` can be thought of as a rough approximation of how many bytes the
transaction occupies in a compressed batch.

`(compressedTxSize) * (16*l1BaseFee*lBaseFeeScalar + l1BlobBaseFee*l1BlobBaseFeeScalar) / 1e6`

The precise cost function used by Ecotone at the top of this section preserves precision under
integer arithmetic by postponing the inner division by 16 until the very end.

[4844-gas]: https://github.com/ethereum/EIPs/blob/master/EIPS/eip-4844.md#gas-accounting

The two base fee values and their respective scalars can be accessed in two interchangeable ways:

- read from the deposited L1 attributes (`l1BaseFeeScalar`, `l1BlobBaseFeeScalar`, `basefee`,
  `blobBaseFee`) of the current L2 block
- read from the L1 Block Info contract (`0x4200000000000000000000000000000000000015`)
  - using the respective solidity getter functions
  - using direct storage-reads:
    - basefee `uint256` in slot `1`
    - blobBaseFee `uint256` in slot `7`
    - l1BaseFeeScalar big-endian `uint32` slot `3` at offset `12`
    - l1BlobBaseFeeScalar big-endian `uint32` in slot `3` at offset `8`

## Engine API

### `engine_forkchoiceUpdatedV2`

This updates which L2 blocks the engine considers to be canonical (`forkchoiceState` argument),
and optionally initiates block production (`payloadAttributes` argument).

Within the rollup, the types of forkchoice updates translate as:

- `headBlockHash`: block hash of the head of the canonical chain. Labeled `"unsafe"` in user JSON-RPC.
   Nodes may apply L2 blocks out of band ahead of time, and then reorg when L1 data conflicts.
- `safeBlockHash`: block hash of the canonical chain, derived from L1 data, unlikely to reorg.
- `finalizedBlockHash`: irreversible block hash, matches lower boundary of the dispute period.

To support rollup functionality, one backwards-compatible change is introduced
to [`engine_forkchoiceUpdatedV2`][engine_forkchoiceUpdatedV2]: the extended `PayloadAttributesV2`

#### Extended PayloadAttributesV2

[`PayloadAttributesV2`][PayloadAttributesV2] is extended to:

```js
PayloadAttributesV2: {
    timestamp: QUANTITY
    random: DATA (32 bytes)
    suggestedFeeRecipient: DATA (20 bytes)
    withdrawals: array of WithdrawalV1
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
This is equivalent to the `transactions` field in [`ExecutionPayloadV2`][ExecutionPayloadV2]

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
so if `engine_forkchoiceUpdatedV2` returns `STATUS_INVALID` it is because a batched transaction is invalid.

The `gasLimit` is optional w.r.t. compatibility with L1, but required when used as rollup.
This field overrides the gas limit used during block-building.
If not specified as rollup, a `STATUS_INVALID` is returned.

[rollup-driver]: rollup-node.md

### `engine_forkchoiceUpdatedV3`

See [`engine_forkchoiceUpdatedV2`](#engine_forkchoiceUpdatedV2) for a description of the forkchoice updated method.

To support rollup functionality, one backwards-compatible change is introduced
to [`engine_forkchoiceUpdatedV3`][engine_forkchoiceUpdatedV3]: the extended `PayloadAttributesV3`

#### Extended PayloadAttributesV3

[`PayloadAttributesV3`][PayloadAttributesV3] is extended to:

```js
PayloadAttributesV3: {
    timestamp: QUANTITY
    random: DATA (32 bytes)
    suggestedFeeRecipient: DATA (20 bytes)
    withdrawals: array of WithdrawalV1
    parentBeaconBlockRoot: DATA (32 bytes)
    transactions: array of DATA
    noTxPool: bool
    gasLimit: QUANTITY or null
}
```

The requirements of this object are the same as extended [`PayloadAttributesV2`][#extended-payloadattributesv2] with
the addition of `parentBeaconBlockRoot` which is the parent beacon block root from the L1 origin block of the L2 block.

The `parentBeaconBlockRoot` must be nil for Bedrock/Canyon/Delta payloads.
Starting at Ecotone, the `parentBeaconBlockRoot` must be set to the L1 origin `parentBeaconBlockRoot`,
or a zero `bytes32` if the Dencun functionality with `parentBeaconBlockRoot` is not active on L1.

### `engine_newPayloadV2`

No modifications to [`engine_newPayloadV2`][engine_newPayloadV2].
Applies a L2 block to the engine state.

### `engine_newPayloadV3`

[`engine_newPayloadV3`][engine_newPayloadV3] applies an Ecotone L2 block to the engine state. There are no
modifications to this API. The additional parameters should be set as follows:

- `expectedBlobVersionedHashes` MUST be an empty array.
- `parentBeaconBlockRoot` MUST be the parent beacon block root from the L1 origin block of the L2 block.

### `engine_getPayloadV2`

No modifications to [`engine_getPayloadV2`][engine_getPayloadV2].
Retrieves a payload by ID, prepared by `engine_forkchoiceUpdatedV2` when called with `payloadAttributes`.

### `engine_getPayloadV3`

[`engine_getPayloadV3`][engine_getPayloadV3] retrieves a payload by ID, prepared by `engine_forkchoiceUpdatedV3`
when called with `payloadAttributes`.

#### Extended Response

The [response][GetPayloadV3Response] is extended to:

```js
{
    executionPayload: ExecutionPayload
    blockValue: QUANTITY
    blobsBundle: BlobsBundle
    shouldOverrideBuilder: BOOLEAN
    parentBeaconBlockRoot: DATA (32 bytes)
}
```

[GetPayloadV3Response]: https://github.com/ethereum/execution-apis/blob/main/src/engine/cancun.md#response-2

For Bedrock and Canyon `parentBeaconBlockRoot` MUST be nil and in Ecotone it MUST be set to the parentBeaconBlockRoot
from the L1 Origin block of the L2 block.

### `engine_signalSuperchainV1`

Optional extension to the Engine API. Signals superchain information to the Engine:
V1 signals which protocol version is recommended and required.

Types:

```javascript
SuperchainSignal: {
    recommended: ProtocolVersion
    required: ProtocolVersion
}
```

`ProtocolVersion`: encoded for RPC as defined in
[Protocol Version format specification](./superchain-upgrades.md#protocol-version-format).

Parameters:

- `signal`: `SuperchainSignal`, the signaled superchain information.

Returns:

- `ProtocolVersion`: the latest supported OP-Stack protocol version of the execution engine.

The execution engine SHOULD warn the user when the recommended version is newer than
the current version supported by the execution engine.

The execution engine SHOULD take safety precautions if it does not meet the required protocol version.
This may include halting the engine, with consent of the execution engine operator.

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
   - Bedrock / Canyon / Delta Payloads
      - [`engine_newPayloadV2`][engine_newPayloadV2] is called with latest L2 block received from P2P.
      - [`engine_forkchoiceUpdatedV2`][engine_forkchoiceUpdatedV2] is called with the current
        `unsafe`/`safe`/`finalized` L2 block hashes.
   - Ecotone Payloads
      - [`engine_newPayloadV3`][engine_newPayloadV3] is called with latest L2 block received from P2P.
      - [`engine_forkchoiceUpdatedV3`][engine_forkchoiceUpdatedV3] is called with the current
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
   as outlined in the [rollup node spec].

[rollup node spec]: rollup-node.md

## Ecotone: disable Blob-transactions

[EIP-4844] introduces Blob transactions: featuring all the functionality of an [EIP-1559] transaction,
plus a list of "blobs": "Binary Large Object", i.e. a dedicated data type for serving Data-Availability as base-layer.

With the Ecotone upgrade, all Cancun L1 execution features are enabled, with [EIP-4844] as exception:
as a L2, the OP-Stack does not serve blobs, and thus disables this new transaction type.

EIP-4844 is disabled as following:

- Transaction network-layer announcements, announcing blob-type transactions, are ignored.
- Transactions of the blob-type, through the RPC or otherwise, are not allowed into the transaction pool.
- Block-building code does not select EIP-4844 transactions.
- An L2 block state-transition with EIP-4844 transactions is invalid.

The [BLOBBASEFEE opcode](https://eips.ethereum.org/EIPS/eip-7516) is present but its semantics are
altered because there are no blobs processed by L2. The opcode will always push a value of 1 onto
the stack.

## Ecotone: Beacon Block Root

[EIP-4788] introduces a "beacon block root" into the execution-layer block-header and EVM.
This block root is an [SSZ hash-tree-root] of the consensus-layer contents of the previous consensus block.

With the adoption of [EIP-4399] in the Bedrock upgrade the OP-Stack already includes the `PREVRANDAO` of L1.
And thus with [EIP-4788] the L1 beacon block root is made available.

For the Ecotone upgrade, this entails that:

- The `parent_beacon_block_root` of the L1 origin is now embedded in the L2 block header.
- The "Beacon roots contract" is deployed at Ecotone upgrade-time, or embedded at genesis if activated at genesis.
- The block state-transition process now includes the same special beacon-block-root EVM processing as L1 ethereum.

[SSZ hash-tree-root]: https://github.com/ethereum/consensus-specs/blob/dev/ssz/simple-serialize.md#merkleization
[EIP-4399]: https://eips.ethereum.org/EIPS/eip-4399
[EIP-4788]: https://eips.ethereum.org/EIPS/eip-4788
[EIP-4844]: https://eips.ethereum.org/EIPS/eip-4844
[eip-1559]: https://eips.ethereum.org/EIPS/eip-1559
[eip-2028]: https://eips.ethereum.org/EIPS/eip-2028
[eip-2718]: https://eips.ethereum.org/EIPS/eip-2718
[eip-2718-transactions]: https://eips.ethereum.org/EIPS/eip-2718#transactions
[exec-api-data]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#structures
[l1-api-spec]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md
[PayloadAttributesV3]: https://github.com/ethereum/execution-apis/blob/cea7eeb642052f4c2e03449dc48296def4aafc24/src/engine/cancun.md#payloadattributesv3
[PayloadAttributesV2]: https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#PayloadAttributesV2
[ExecutionPayloadV1]: https://github.com/ethereum/execution-apis/blob/769c53c94c4e487337ad0edea9ee0dce49c79bfa/src/engine/specification.md#ExecutionPayloadV1
[engine_forkchoiceUpdatedV3]: https://github.com/ethereum/execution-apis/blob/cea7eeb642052f4c2e03449dc48296def4aafc24/src/engine/cancun.md#engine_forkchoiceupdatedv3
[engine_forkchoiceUpdatedV2]: https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_forkchoiceupdatedv2
[engine_newPayloadV2]: https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_newpayloadv2
[engine_newPayloadV3]: https://github.com/ethereum/execution-apis/blob/cea7eeb642052f4c2e03449dc48296def4aafc24/src/engine/cancun.md#engine_newpayloadv3
[engine_getPayloadV2]: https://github.com/ethereum/execution-apis/blob/584905270d8ad665718058060267061ecfd79ca5/src/engine/shanghai.md#engine_getpayloadv2
[engine_getPayloadV3]: https://github.com/ethereum/execution-apis/blob/a0d03086564ab1838b462befbc083f873dcf0c0f/src/engine/cancun.md#engine_getpayloadv3
[HEX value encoding]: https://eth.wiki/json-rpc/API#hex-value-encoding
[JSON-RPC-API]: https://github.com/ethereum/execution-apis
