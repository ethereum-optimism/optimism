# Synchronization Spec

The transaction data can be indexed from either L1 or L2. This data is
then used for execution which generates state. The benefit of syncing from
L2 is that the data will be synced before the transactions are batch submitted.
The benefit of syncing from L1 is a guarantee that the transactions are final
relative to L2, assuming no fraud proof.

## Indexing Data From Layer 1

We need a reliable stream of data from our Layer 1 smart contracts.
This doc specifies what data we need and how we're getting it.
See [Appendix](#appendix) for a list of data structures.

### L1 Transaction Indexer API

#### getEnqueuedTransactionByIndex

```ts
function getEnqueuedTransactionByIndex(index: number): EnqueuedTransaction;
```

#### getLatestEnqueuedTransaction

```ts
function getLatestEnqueuedTransaction(): EnqueuedTransaction;
```

#### getTransactionByIndex

```ts
function getTransactionByIndex(index: number): Transaction;
```

#### getLatestTransaction

```ts
function getLatestTransaction(): Transaction;
```

#### getStateRootByIndex

```ts
function getStateRootByIndex(index: number): StateRoot;
```

#### getLatestStateRoot

```ts
function getLatestStateRoot(): StateRoot;
```

### Requirements

- Must get data within X amount of time (?)
- Must not get reorg'd out of existence.

### Retrieving Relevant Data

We need to retrieve the following information reliably:

- All transactions enqueued for inclusion in the CanonicalTransactionChain in the order in which they were enqueued.
- All transactions included in the CanonicalTransactionChain in the order in which they were included.
- All state roots included in the StateCommitmentChain in the order in which they were included.

All relevant data can be retrieved by parsing data from the following functions:

- `OVM_CanonicalTransactionChain.enqueue`
- `OVM_CanonicalTransactionChain.appendQueueBatch`
- `OVM_CanonicalTransactionChain.appendSequencerBatch`
- `OVM_StateCommitmentChain.appendStateBatch`

#### Enqueued Transactions

Transactions are "enqueued" when users make calls to `OVM_CanonicalTransactionChain.enqueue`.
Calls to this function can be detected by searching for [`TransactionEnqueued`](#transactionenqueued) events.
All relevant transaction data can be pulled out of the event, here's a pseudocode function for doing so:

```ts
function parseTransactionEnqueuedEvent(
    event: TransactionEnqueued,
): EnqueuedTransaction {
    return {
        queueIndex: event.queueIndex,
        timestamp: event.timestamp,
        blockNumber: getBlockNumber(event),
        l1QueueOrigin: QueueOrigin.L1TOL2_QUEUE,
        l1TxOrigin: event.l1TxOrigin
        entrypoint: event.target,
        gasLimit: event.gasLimit,
        data: event.data
    }
}
```

So the process of parsing and indexing these transactions is pretty straightforward:

1. Listen to [`TransactionEnqueued`](#transactionenqueued) events.
2. Parse each found event into `EnqueuedTransaction` structs.
3. Store each event based on their `queueIndex` field.

#### Transactions (via appendQueueBatch)

::: tip Note
`appendQueueBatch` is currently disabled on mainnet.
:::

Transactions can be inserted into the "canonical transaction chain" via either `appendQueueBatch` or `appendSequencerBatch`.
`appendQueueBatch` is used to move enqueued transactions into the canonical set of transactions.
Until this is done, these enqueued transactions are not considered part of the L2 history.

Whenever `appendQueueBatch` is called, a [`TransactionBatchAppended`](#transactionbatchappended) event is emitted followed by a [`QueueBatchAppended`](#queuebatchappended) event in the same transaction (such that the index of the second event is equal to the index of the first event plus one).
These events do **not** include the complete forms of the various transactions that were "appended" by this function call -- only pointers to the corresponding queue elements.
As a result, we're required to pull this information from a combination of the events and the `EnqueuedTransaction` objects previously parsed from `enqueue`.

More pseudocode for parsing this list of transactions:

```ts
function parseQueueBatchAppendedEvent(
    event: QueueBatchAppended
): Transaction[] {
    // Get the `TransactionBatchAppended` event. Really should be turned into a
    // single event to avoid having to do this extra network request.
    const event2: TransactionBatchAppended = getEventByIndex(
        getEventIndex(event) - 1
    )

    const transactions: Transaction[] = []
    for (let i = 0; i < event.numQueueElements) {
        // Note that this places an assumption on how events are parsed. This
        // only works if enqueued transactions are parsed before
        // `appendQueueBatch` events.
        const enqueuedTransaction: EnqueuedTransaction = getEnqueuedTransactionByIndex(
            event.startingQueueIndex + i
        )

        transactions.push({
            l1QueueOrigin: QueueOrigin.L1TOL2_QUEUE,
            timestamp: enqueuedTransaction.timestamp,
            blockNumber: enqueuedTransaction.blockNumber,
            l1TxOrigin: enqueuedTransaction.l1TxOrigin,
            entrypoint: enqueuedTransaction.entrypoint,
            gasLimit: enqueuedTransaction.gasLimit,
            data: enqueuedTransaction.data
        })
    }

    // TODO: Add parsing batches.

    return transactions
}
```

#### Transactions (via appendSequencerBatch)

`appendSequencerBatch` is the other method by which transactions can be inserted into the canonical transaction chain.
`appendSequencerBatch` makes use of a custom encoding scheme for efficiency reasons and does not have any explicit parameters.

The function internally parses `calldata` directly using the following format:

- Bytes 0-3 (4 bytes) of `calldata` are the 4 byte function selector derived from `keccak256("appendSequencerBatch()")`. Just skip these four bytes.
- Bytes 4-8 (5 bytes; `uint40`) describe the index of the "canonical transaction chain" that this batch of transactions expects to follow.
- Bytes 9-11 (3 bytes; `uint24`) are the total number of elements that the sequencer wants to append to the chain.
- Bytes 12-14 (3 bytes; `uint24`) are the total number of "batch contexts," effectively timestamp/block numbers to be assigned to given sets of transactions.
- After byte 14, we have a series of encoded "batch contexts." Each batch context is exactly **16 bytes**. The number of contexts comes from bytes 12-14, as described above. Each context has the following structure:
  - Bytes 0-2 (3 bytes; `uint24`) are the number of sequencer transactions that will utilize this batch context.
  - Bytes 3-5 (3 bytes; `uint24`) are the number of _queue_ transactions that will be inserted into the chain after these sequencer transactions.
  - Bytes 6-10 (5 bytes; `uint40`) are the timestamp that will be assigned to these sequencer transactions.
  - Bytes 11-15 (5 bytes; `uint40`) are the block number that will be assigned to these sequencer transactions.
- After the batch context section, we have a series of dynamically sized transactions. Each transaction consists of the following information:
  - Bytes 0-2 (3 bytes: `uint24`) are the total size of the coming transaction data in bytes.
  - Some arbitrary data of a length equal to that described by the first three bytes.

We can represent the input as an object roughly equivalent to the following json-ish thing:

```ts
interface AppendSequencerBatchParams {
    sighash: 4 bytes,
    shouldStartAtElement: 5 bytes,
    totalElementsToAppend: 3 bytes,
    numContexts: 3 bytes,
    contexts: Array<{
        numSequencedTransactions: 3 bytes,
        numSubsequentQueueTransactions: 3 bytes,
        ctxTimestamp: 5 bytes,
        ctxBlockNumber: 5 bytes
    }>,
    transactions: Array<{
        txDataLength: 3 bytes,
        txData: txDataLength bytes
    }>
}
```

Decoding function (in pseudocode):

```ts
function decode(calldata: bytes): AppendSequencerBatchParams {
    const sighash = calldata[0:4]
    const shouldStartAtElement = uint40(calldata[4:9])
    const totalElementsToAppend = uint24(calldata[9:12])
    const numContexts = uint24(calldata[12:15])

    let ptr = 15
    const contexts = []
    for (let i = 0; i < numContexts; i++) {
        contexts.push({
            numSequencedTransactions: uint24(calldata[ptr:ptr+3])
            numSubsequentQueueTransactions: uint24(calldata[ptr+3:ptr+6]),
            ctxTimestamp: uint40(calldata[ptr+6:ptr+11]),
            ctxBlockNumber: uint40(calldata[ptr+11:ptr+16])
        })

        ptr = ptr + 16
    }

    const transactions = []
    while (ptr < length(calldata)) {
        const txDataLength = uint24(calldata[ptr:ptr+3])
        transactions.push({
            txDataLength: txDataLength,
            txData: calldata[ptr+3:ptr+3+txDataLength]
        })

        ptr = ptr + 3 + txDataLength
    }

    return {
        sighash: sighash,
        shouldStartAtElement: shouldStartAtElement,
        totalElementsToAppend: totalElementsToAppend,
        numContexts: numContexts,
        contexts: contexts,
        transactions: transactions,
    }
}
```

Encoding function (in pseudocode):

```ts
function encode(params: AppendSequencerBatchParams): bytes {
    let calldata = bytes()

    calldata[0:4] = bytes4(params.sighash)
    calldata[4:9] = bytes5(params.shouldStartAtElement)
    calldata[9:12] = bytes3(params.totalElementsToAppend)
    calldata[12:15] = bytes3(params.numContexts)

    let ptr = 15
    for (const context of params.contexts) {
        calldata[ptr:ptr+3] = bytes3(context.numSequencedTransactions)
        calldata[ptr+3:ptr+6] = bytes3(context.numSubsequentQueueTransactions)
        calldata[ptr+6:ptr+11] = bytes5(context.ctxTimestamp)
        calldata[ptr+11:ptr+16] = bytes5(context.ctxBlockNumber)

        ptr = ptr + 16
    }

    for (const transaction of params.transactions) {
        const txDataLength = transaction.txDataLength
        calldata[ptr:ptr+3] = bytes3(txDataLength)
        calldata[ptr+3:ptr+3+txDataLength] = bytes(transaction.data)

        ptr = ptr + 3 + txDataLength
    }

    return calldata
}
```

When the sequencer calls `appendQueueBatch`, `contexts` are processed one by one.
For each context, we first append `numSequencedTransactions` which are popped off of the `transactions` array.
Each of these transactions have the following form:

```solidity
Transaction({
    timestamp:     context.ctxTimestamp,
    blockNumber:   context.ctxBlockNumber,
    l1QueueOrigin: QueueOrigin.SEQUENCER_QUEUE,
    l1TxOrigin:    0x0000000000000000000000000000000000000000,
    entrypoint:    0x4200000000000000000000000000000000000005,
    gasLimit:      OVM_ExecutionManager.getMaxTransactionGasLimit(),
    data:          tx.txData,
})
```

We next pull `numSubsequentQueueTransactions` transactions in from the queue (by reference to the queue).
This process is repeated for every provided `context`.

`appendSequencerBatch` can be detected (and parsed) by looking for [`SequencerBatchAppended`](#sequencerbatchappended) events.
Each of these events will always be immediately preceeded by a [`TransactionBatchAppended`](#transactionbatchappended) event.
Somewhat like with `appendQueueBatch`, we'll have to carefully pull all of the relevant information out of these two events and previously parsed `EnqueuedTransactions`.

Notably, we also need access to the `calldata` sent to `appendSequencerBatch`.
We can retrieve this data by making a call to [`debug_traceTransaction`](https://geth.ethereum.org/docs/rpc/ns-debug#debug_tracetransaction) (or an equivalent endpoint that exposes call traces).
If the client does not have access to the `debug_traceTransaction` endpoint, then the `calldata` can only be retrieved if `appendSequencerBatch` is called directly by an externally owned account (because then `calldata === transaction.input`, which is easily [accessible via the standard API](https://eth.wiki/json-rpc/API#eth_getTransactionByHash)).

When the client detects a `SequencerBatchAppended` event, they should pull the preceeding `TransactionBatchAppended` event.
Then they should retrieve the `calldata` and decode it using the above decoding scheme.
Client **should** validate input under the assumption that data from the L1 node is not reliable.

Finally, pseudocode for parsing the event:

```ts
function parseSequencerBatchAppendedEvent(
  event: QueueBatchAppended
): Transaction[] {
  // Get the `TransactionBatchAppended` event. Really should be turned into a
  // single event to avoid having to do this extra network request.
  const event2: TransactionBatchAppended = getEventByIndex(
    getEventIndex(event) - 1
  );

  const calldata: bytes = getCalldataByTransaction(getTransaction(event));

  const params: AppendSequencerBatchParams = decode(calldata);

  let sequencerTransactionCount = 0;
  let queueTransactionCount = 0;
  const transactions: Transaction[] = [];
  for (const context of params.contexts) {
    for (let i = 0; i < context.numSequencerTransactions; i++) {
      transactions.push({
        l1QueueOrigin: QueueOrigin.SEQUENCER_QUEUE,
        timestamp: context.ctxTimestamp,
        blockNumber: context.ctxBlockNumber,
        l1TxOrigin: 0x0000000000000000000000000000000000000000,
        entrypoint: 0x4200000000000000000000000000000000000005,
        gasLimit: OVM_ExecutionManager.getMaxTransactionGasLimit(),
        data: params.transactions[sequencerTransactionCount],
      });

      sequencerTransactionCount = sequencerTransactionCount + 1;
    }

    for (let i = 0; i < context.numSubsequentQueueTransactions; i++) {
      // Note that this places an assumption on how events are parsed. This
      // only works if enqueued transactions are parsed before
      // `appendQueueBatch` events.
      const enqueuedTransaction: EnqueuedTransaction =
        getEnqueuedTransactionByIndex(
          event.startingQueueIndex + queueTransactionCount
        );

      transactions.push({
        l1QueueOrigin: QueueOrigin.L1TOL2_QUEUE,
        timestamp: enqueuedTransaction.timestamp,
        blockNumber: enqueuedTransaction.blockNumber,
        l1TxOrigin: enqueuedTransaction.l1TxOrigin,
        entrypoint: enqueuedTransaction.entrypoint,
        gasLimit: enqueuedTransaction.gasLimit,
        data: enqueuedTransaction.data,
      });

      queueTransactionCount = queueTransactionCount + 1;
    }
  }

  // TODO: Add parsing batches.

  return transactions;
}
```

## Appendix

### Enums

#### QueueOrigin

```solidity
enum QueueOrigin {
  SEQUENCER_QUEUE,
  L1TOL2_QUEUE
}

```

### Structs

#### Transaction

```solidity
struct Transaction {
  QueueOrigin l1QueueOrigin;
  uint256 timestamp;
  uint256 blockNumber;
  address l1TxOrigin;
  address entrypoint;
  uint256 gasLimit;
  bytes data;
}

```

#### EnqueuedTransaction

```solidity
struct EnqueuedTransaction {
  QueueOrigin l1QueueOrigin;
  uint256 timestamp;
  uint256 blockNumber;
  address l1TxOrigin;
  address entrypoint;
  uint256 gasLimit;
  bytes data;
  uint256 queueIndex;
}

```

### Events

#### TransactionEnqueued

```solidity
event TransactionEnqueued(
    address l1TxOrigin,
    address target,
    uint256 gasLimit,
    bytes   data,
    uint256 queueIndex,
    uint256 timestamp,
    uint256 blockNumber
);
```

#### TransactionBatchAppended

```solidity
event TransactionBatchAppended(
    uint256 indexed batchIndex,
    bytes32 batchRoot,
    uint256 batchSize,
    uint256 prevTotalElements,
    bytes   extraData
);
```

#### SequencerBatchAppended

```solidity
event SequencerBatchAppended(
    uint256 startingQueueIndex,
    uint256 numQueueElements,
    uint256 totalElements
);
```

#### QueueBatchAppended

```solidity
event QueueBatchAppended(
    uint256 startingQueueIndex,
    uint256 numQueueElements,
    uint256 totalElements
);
```

## Indexing Data From Layer 2

It is possible to sync data from L2 and expose the same API
