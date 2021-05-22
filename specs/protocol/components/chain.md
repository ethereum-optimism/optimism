# Chains

This document describes the Transaction Queue, the State Commitment Chain, the Canonical Transaction Chain.

- [Chains](#chains)
  - [Transaction Queue](#transaction-queue)
  - [Structure of the Transaction Queue](#structure-of-the-transaction-queue)
    - [Transaction Queue Processes](#transaction-queue-processes)
      - [Appending to the Transaction Queue](#appending-to-the-transaction-queue)
  - [Canonical Transaction Chain](#canonical-transaction-chain)
    - [Canonical Transaction Chain Structure](#canonical-transaction-chain-structure)
    - [Verifying transaction inclusion in the CTC](#verifying-transaction-inclusion-in-the-ctc)
    - [Updating the CTC](#updating-the-ctc)
      - [Appending to the CTC by `appendSequencerBatch()`](#appending-to-the-ctc-by-appendsequencerbatch)
      - [Transaction ordering requirements](#transaction-ordering-requirements)
        - [Force Inclusion Period rules](#force-inclusion-period-rules)
        - [Context properties](#context-properties)
      - [Appending to the CTC by `appendQueueBatch()`](#appending-to-the-ctc-by-appendqueuebatch)
        - [`appendQueueBatch()` requirements](#appendqueuebatch-requirements)
  - [State Commitment Chain](#state-commitment-chain)
    - [SCC Structure](#scc-structure)
    - [Verifying state root inclusion in the SCC](#verifying-state-root-inclusion-in-the-scc)
    - [Updating the SCC](#updating-the-scc)
      - [Appending batches to the SCC](#appending-batches-to-the-scc)
        - [`appendStateBatch()` requirements](#appendstatebatch-requirements)
      - [Deleting batches from the SCC](#deleting-batches-from-the-scc)
        - [`deleteStateBatch()` requirements](#deletestatebatch-requirements)

## Transaction Queue

The Transaction Queue is an append-only array of transaction data which MUST eventually be included in the Canonical Transaction Chain. The Transaction Queue itself does not describe the L2 state, but it serves as a source of transaction inputs to the CTC (the other source being the Sequencer).

## Structure of the Transaction Queue

Two `bytes32` entries are stored for each Queue transaction:

1. The first entry is `transactionHash` defined in solidity as:

```jsx
bytes32 transactionHash = keccak256(
  abi.encode(
      msg.sender,
      _target,
      _gasLimit,
      _data
  )
);
```

2. The second entry is `timestampAndBlockNumber`, which records the `TIMESTAMP` and `NUMBER` values in the EVM at write time. This entry is defined in solidity as:

```jsx
bytes32 timestampAndBlockNumber;
assembly {
   timestampAndBlockNumber := timestamp()
   timestampAndBlockNumber := or(timestampAndBlockNumber, shl(40, number()))
}
```

There is a [`QueueElement`](../data-structures.md#queueelement) type, which is also sometimes used to represent this data.

### Transaction Queue Processes

The Transaction Queue is append-only, thus the only allowed update operation is appending to it.

The Queue has two distinct purposes:

1. Resistance against a censorious Sequencer (see [Actors and Roles](../actors-and-roles.md)).
2. Sending messages (ie. deposits) from L1 to L2 (see [Cross Domain Messaging](./cross-domain-messaging.md)).

#### Appending to the Transaction Queue

Any account may append to the Transaction Queue by calling the CTC's `enqueue()` function:

```jsx
function enqueue(
    address _target,
    uint256 _gasLimit,
    bytes _data
)
```

Where the parameters are:

- `address _target`: Target L2 contract to send the transaction to.
- `uint256 _gasLimit`: Gas limit for the enqueued L2 transaction.
- `bytes _data`: Arbitrary calldata for the enqueued L2 transaction.

## Canonical Transaction Chain

The Canonical Transaction Chain (CTC) is an append-only array of transactions which MUST be processed in-order to determine and verify the L2 state. The transaction data is compressed into batches to reduce storage costs.

Note: there are no blocks here, the CTC is just an ordered list of transactions.

### Canonical Transaction Chain Structure

Entries in the CTC are of type `bytes32 batchHeaderHash`, defined as the hash of the elements of a [`ChainBatchHeader`](../data-structures.md#chainbatchheader) (note that the `batchIndex` is not included).

```jsx
keccak256(
  abi.encode(
    _batchHeader.batchRoot,
    _batchHeader.batchSize,
    _batchHeader.prevTotalElements,
    _batchHeader.extraData
  )
);
```

Additionally, the CTC maintains its 'global context' in a `bytes27 latestBatchContext`, which encodes the fields of the [Extra Data](../data-structures#extra-data) structures as follows:

```jsx
bytes27 extraData;
assembly {
   extraData := _totalElements
   extraData := or(extraData, shl(40, _nextQueueIndex))
   extraData := or(extraData, shl(80, _timestamp))
   extraData := or(extraData, shl(120, _blockNumber))
   extraData := shl(40, extraData)
}
```

### Verifying transaction inclusion in the CTC

The CTC's `verifyTransaction` function returns a boolean indicating whether a transaction is included in the chain:

```jsx
function verifyTransaction(
    Transaction _transaction,
    TransactionChainElement _txChainElement,
    ChainBatchHeader _batchHeader,
    ChainInclusionProof _inclusionProof
)
  returns(
    bool
  )
```

Where the parameters are:

- `Transaction _transaction`: Transaction to verify.
- `TransactionChainElement _txChainElement`: Transaction chain element corresponding to the transaction.
- `ChainBatchHeader _batchHeader`: Header of the batch the transaction was included in.
- `ChainInclusionProof _inclusionProof`: Inclusion proof for the provided transaction chain element.

<!-- TODO: we should add a note somewhere in this section explaining the implicit structure underneath these batches, i.e. the actual leaves of the trees. -->

### Updating the CTC

The Transaction Queue is append-only, thus the only allowed update operation is appending to it. There are two methods by which this can be done.

1. `appendSequencerBatch()`
2. `appendQueueBatch()`

#### Appending to the CTC by `appendSequencerBatch()`

The Sequencer appends transactions to the chain in batches by calling the CTC's `appendSequencerBatch()` function, defined in solidity as:

```jsx
function appendSequencerBatch()
```

The data provided MUST conform to a custom encoding scheme (which is used for efficiency reasons). The scheme is described [here](../../l2-geth/transaction-indexer.md#transactions-via-appendsequencerbatch).

The `BatchContext` data provided by the Sequencer will be used to determine the ordering of transactions in the CTC:

- First `BatchContext.numSequencedTransactions` are added from the `_transactionDataFields`
- Then `BatchContext.numSubsequentQueueTransactions` are added from the `queue`.

This process is repeated until [`totalElements`](../data-structures.md#extra-data) transactions have been appended.

#### Transaction ordering requirements

The following constraints MUST be imposed on the ordering of Sequencer and Queue transactions (for the purpose of preventing a malicious Sequencer attempting to censor transaction):

##### Force Inclusion Period rules

The `Force Inclusion Period` is a storage variable defined in the CTC. It is expected to be on the order to 10 to 60 minutes.

1. If any queue elements are older than the Force Inclusion Period, they must be appended to the chain before any Sequencer transactions.
2. The Sequencer MUST not be able to insert Sequencer Transactions older than Force Inclusion Period.

##### Context properties

`BatchContext.blockNumber` and `BatchContext.timestamp` MUST be:

- monotonically increasing
- less than or equal to the L1 `blockNumber` and `timestamp` when `appendSequencerBatch()` is called
- less than or equal to the `blockNumber` and `timestamp` on all `QueueElements`

An important high level property which emerges from these rules is that Queue transactions will always have the same timestamp/blocknumber on L2 as the L1 block during which they were enqueued.

_Note: There may be some implicit requirements missing from here_.

#### Appending to the CTC by `appendQueueBatch()`

Any account MAY append transactions from the Queue to the CTC by calling `appendQueueBatch()`:

```jsx
function appendQueueBatch(
    uint256 _numQueuedTransactions
)
```

Where the parameter is:

- `uint256 _numQueuedTransactions`: the number of transactions from the queue to be appended to the CTC.

##### `appendQueueBatch()` requirements

Transactions MUST have been added to the Queue earlier than `now - forceInclusionPeriod`. If any transactions are newer, `appendQueueBatch()` will revert.

## State Commitment Chain

The State Commitment Chain (SCC) is an array of StateCommitmentBatches, where each batch is a Merkle root derived from an array of state roots.

In practice the SCC should be append-only, but in the event of a successful fraud proof, the state may be rolled back to the most recent honest state.

### SCC Structure

The SCC is an array of [`ChainBatchHeader`s](../data-structures.md#chainbatchheader).

### Verifying state root inclusion in the SCC

The SCC's `verifyTransaction` function returns a boolean indicating whether a transaction is included in the chain:

```jsx
function verifyStateCommitment(
    bytes32 _element,
    ChainBatchHeader _batchHeader,
    ChainInclusionProof _proof
)
    returns (
        bool _verified
      );
```

Where the parameters are:

- `bytes32 _element`: Hash of the element to verify a proof for.
- `ChainBatchHeader _batchHeader`: Header of the batch in which the element was included.
- `ChainInclusionProof _proof`: Merkle inclusion proof for the element.

### Updating the SCC

#### Appending batches to the SCC

An account MAY act as a Proposer, call the SCC's `appendStateBatch()` function to commit to L2 state roots (if and only if it is approved by the Bond Manager contract).

_At this time, the Bond Manager will only approve the Sequencer. In the future this will be extended to any account which has deposited the required collateral._

```jsx
function appendStateBatch(
    bytes32[] _batch,
    uint256 _shouldStartAtElement
)
```

The logic of `appendStateBatch()` will compute the Merkle root of the batches and append it to the SCC.

##### `appendStateBatch()` requirements

- The batch may not be empty, at least one new state root must be added
- The `OVM_BondManager.isCollateralized(msg.sender)` must return `true`
- The resulting number of elements in the SCC (ie. all state commitments, not batches) must be less than or equal to the number of elements (ie. transactions) in the CTC.

#### Deleting batches from the SCC

In the event of a successful fraud proof, the Fraud Verifier contract may delete the State Batch which contains the fraudulent state root, by calling `deleteStateBatch()`:

```jsx
function deleteStateBatch(
    ChainBatchHeader _batchHeader
)
```

Where the parameters are:

- `ChainBatchHeader _batchHeader`: The header of the state batch to delete.

Note that this may result in valid state commitments prior to the fraudulent state being deleted. This is OK, the next honest proposer can resubmit them.

##### `deleteStateBatch()` requirements

- The caller MUST be the Fraud Verifier contract
- The fraud proof window must not have passed at runtime
- The batch header must be valid
