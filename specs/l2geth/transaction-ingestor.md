# Transaction Ingestor

This spec is written in Python pseudocode. It may differ slightly from actual Python, but it should be close enough to
implement correctly in any language. It is currently implemented as a fork of `go-ethereum` and includes some implementation
specific details.

### Types

The types used throughout this document are listed below.

#### `Transaction`

```python
class Transaction:
    meta: TransactionMeta
    nonce: int
    to: Address
    data: bytes
    gas: int
    gasPrice: int
    signature: bytes[65]
```

The `Transaction` is similar to an `ethereum.Transaction` but includes an extra field `meta`. In the future account abstracted world,
a `Transaction` may just be a typed blob. Note that Optimism already has account abstraction, but it is not a first class citizen
relative to the node software as to maintain a compatible Ethereum RPC API.

#### `TransactionMeta`

```python
class TransactionMeta:
    index: int
    queue_index: int
    l1_timestamp: int
    l1_blocknumber: int
    l1_tx_origin: Address
    queue_origin: QueueOrigin
    type: TransactionType
```

The `TransactionMeta` includes specific Layer Two information that differs from the traditional Ethereum Transaction.
`TransactionMeta.index` being present indicates that this transaction has been sequenced by the sequencer and
its value is equal to the transaction's CTC index or the future CTC index in the case where the transaction has yet to be
batch submitted. `TransactionMeta.queue_index` is the queue index and is only present for L1 to L2 transactions.

#### `QueueOrigin`

```python
class QueueOrigin(enum):
    L1_TO_L2
    SEQUENCER
```

The `QueueOrigin` is a property of the `TransactionMeta` and the value indicates the place that the transaction originated
from. An `L1_TO_L2` transaction originates from Layer One and a `SEQUENCER` transaction originates from the sequencer.

#### `TransactionType`

```python
class TransactionType(enum):
    EIP155
    ETH_SIGN
```

The `TransactionType` is a property of the `TransactionMeta`. The `EIP155` type represents an EIP155 transaction without
a value property. The `ETH_SIGN` type represents an EIP155 like transaction where the signature hash is based on an
abi encoding `eth_sign` signature hash. There is no native value property.

#### `NodeType`

```python
class NodeType(enum):
    VERIFIER
    SEQUENCER
```

The `NodeType` is a property of the `TransactionExecutor` and refers to the permissions of the node in the system. The
`SEQUENCER` can extend the chain by submitting batches and the `VERIFIER` is used to compute state roots to determine
if a fraud proof should be submitted.

#### `Block`

```python
class Block(ethereum.Block):
    transactions: Tuple[ethereum.Transaction]
```

The `Block` is similar to the standard Ethereum Block except that it is defined to have only a single transaction.
A `Tuple` is used to define the fixed size nature of `Block.transactions`.

#### `Blockchain`

```python
class Blockchain(ethereum.Blockchain):
    @abc.abstractmethod
    def get_block_at_index(index: int) -> Block:
        return

    @abc.abstractmethod
    def reorg(index: int):
        return
```

The `Blockchain` represents the underlying blockchain implementation. It is functionally the same as the blockchain
implementation in an Ethereum node implementation. The underlying implementations of `get_block_at_index` and
`reorg` are omitted and assumed to be correct.

#### `Miner`

```python
class Miner(ethereum.Miner):
    @abc.abstractmethod
    def apply_transaction(tx: Transaction) -> bool:
        return
```

The `Miner` represents the underlying `Miner` implementation. It is functionally the same as the `Miner` implementation
in an Ethereum implementation. Note that `apply_transaction` will add a block with a single transaction to the
`Blockchain` after it is called.

#### `TxPool`

```python
class TxPool:
    @abc.abstractmethod
    def validate_tx(tx: Transaction):
        return
```

The `TxPool` represents the underlying `TxPool` implementation. It is only used validate incoming `QueueOrigin.SEQUENCER`
transactions. It is the same validation method that is called in a normal Ethereum node implementation before a
transaction is added to the mempool. It checks that the nonce is correct and that there is enough gas to cover the
instrinsic gas.

#### `Batch`

```python
class Batch:
    index: int
    blockNumber: int
    timestamp: int
    submitter: Address
    size: int
    root: str
    prevTotalElements: int
    extraData: str
```

The `Batch` represents batched elements that have been submitted to the
Canonical Transaction Chain. The same data structure applies for both
transaction batches and state root batches.

#### `TransactionBatch`

```python
class TransactionBatch:
    batch: Batch
    transactions: List[Transaction]
```

#### `Backend`

The backend represents the data transport layer backend. It can be configured to
sync from either L1 or L2

```python
class Backend(enum):
    L1
    L2
```

#### `RollupClient`

```python
class RollupClient:
    @abc.abstractmethod
    def get_enqueue(index: int) -> Transaction:
        return

    @abc.abstractmethod
    def get_latest_enqueue() -> Transaction:
        return

    @abc.abstractmethod
    def get_transaction(index: int, backend: Backend) -> Transaction:
        return

    @abc.abstractmethod
    def get_latest_transaction(backend: Backend) -> Transaction:
        return

    @abc.abstractmethod
    def get_eth_context(index: int) -> EthContext:
        return

    @abc.abstractmethod
    def get_latest_eth_context(index: int) -> EthContext:
        return

    @abc.abstractmethod
    def get_last_confirmed_enqueue() -> Transaction:
        return

    @abc.abstractmethod
    def sync_status() -> SyncStatus:
        return

    @abc.abstractmethod
    def get_latest_tx_batch() -> TransactionBatch:
        return

    @abc.abstractmethod
    def get_tx_batch(index: int) -> TransactionBatch:
        return
```

The `RollupClient` is used to communicate with the transaction indexer. The implementation of the transaction indexer currently
exists as a different process but it may be pulled into the same process in the future.

#### `RollupContext`

```python
class RollupContext:
    current_index: int
    current_queue_index: int
    current_tx_batch_index: int
```

The `RollupContext` consists of the `current_index` and `current_queue_index`. This allows the sequencer to prevent
replaying of transactions as well as safely restarting without double ingesting transactions. The `current_index` corresponds to the
transaction index in the Canonical Transaction Chain and the `current_queue_index` corresponds to the `queue_index` in the Canonical
Transaction Chain. The Canonical Transaction Chain is deployed on Layer One and is the source of truth for the state of the system.
These must be stored in a database.

Note that the `queue_index` refers to the index of the L1 to L2 transactions that have been sent to the `enqueue()` method in the CTC
while the `index` refers to the index of transactions in the CTC that have been appended with `sequencerBatchAppend()`
or `queueBatchAppend()`. After transactions are submitted via one of those methods, they can be considered finalized relative to Layer Two.

#### `EthContext`

```python
class EthContext:
    last_l1_blocknumber: int
    last_l1_timestamp: int
```

The `EthContext` includes the`last_l1_timestamp` and the `last_l1_blocknumber`. They are used for the EVM context at
transaction execution time. These values must be monotonic.
An L1 to L2 transaction must have the same timestamp and blocknumber in its EVM context on L2 as the L1 transaction itself.
The `TransactionExecutor` is also responsible for assigning timestamps to transactions sent directly to the sequencer.
These must be stored in a database.

#### `DB`

```python
class DB:
    @abc.abstractmethod
    def get_current_index() -> int:
        return

    @abc.abstractmethod
    def get_current_queue_index() -> int:
        return

    @abc.abstractmethod
    def get_current_tx_batch_index() -> int:
        return
```

The `DB` is used to persist the `RollupContext`. Its implementation is omitted but it is required for clean
shutdowns of the node.

#### `NodeOptions`

```python
class NodeOptions:
    db: DB
    ctc_deploy_height: int
    type: NodeType
    l1_transaction_indexer_url: str
    replica_transaction_indexer_url: str
```

The `NodeOptions` are the options passed to a node at runtime. It has both a `l1_transaction_indexer_url` as well as a
`replica_transaction_indexer_url` to allow for upgrades when running a `ReplicaSequencer`.

## Node Types + Modes

There are different node types and modes. Each extends the `TransactionExecutor` and has slightly different behavior.

- Sequencer
- Sequencer Replica
- Verifier
- Verifier Replica

### `TransactionExecutor`

The `TransactionExecutor` needs to track both the `RollupContext` and the `EthContext`. The `TransactionExecutor.start` method
is used to start the node and each moded type must implement this method.

```python
import abc

class TransactionExecutor:
    def __init__(self, options):
        self.eth_context = EthContext()
        self.rollup_context = RollupContext()
        self.db = options.db
        self.rollup_context.current_index = self.db.get_current_index()
        self.rollup_context.current_queue_index = self.db.get_current_queue_index()
        self.rollup_context.current_batch_tx_index = options.db.get_current_tx_batch_index()
        self.backend = None
        self.type = None
        self.client = None
        self.ctc_deploy_height = options.ctc_deploy_height
        self.miner = Miner()
        self.blockchain = Blockchain()
        self.tx_pool = TxPool()
        self.poll_interval = options.poll_interval

    @abc.abstractmethod
    def start():
        return

    def set_initial_eth_context():
        if self.eth_context.last_l1_timestamp is None:
            context = self.client.get_eth_context(self.ctc_deploy_height)
            self.eth_context.last_l1_timestamp = context.timestamp
            self.eth_context.last_l1_blocknumber = context.blocknumber
        else:
            block = self.blockchain.get_block_at_index(self.ctc_deploy_height)
            tx = block.transactions[0]
            self.eth_context.last_l1_timestamp = tx.meta.l1_timestamp
            self.eth_context.last_l1_blocknumber = tx.meta.l1_blocknumber

    def set_initial_rollup_context():
        if self.rollup_context.current_enqueue is None:
            tx = self.client.get_last_confirmed_enqueue()
            if tx is not None:
                self.rollup_context.current_enqueue = tx.meta.index
        block = self.blockchain.current_block()
        if block.number - 1 != self.rollup_context.current_index:
            self.rollup_context.current_index = block.number - 1

    def apply_transaction(self, tx):
        if tx.meta.index != None:
            return self.apply_indexed_transaction(tx)
        return self.apply_transaction_to_tip(tx)

    def apply_indexed_transaction(self, tx):
        index = tx.meta.index
        if index == self.rollup_context.current_index + 1:
            return apply_transaction_to_tip(tx)
        if index <= self.rollup_context.current_index:
            return apply_historical_transaction(tx)
        raise Exception("Received future transaction")

    def apply_historical_transaction(self, tx):
        index = tx.meta.index
        block = self.blockchain.get_block_at_index(index)
        if block is None:
            raise Exception("Missing historical block")
        if block.txs[0] is not tx:
            self.blockchain.reorg(tx.meta.index - 1)
            return self.apply_transaction_to_tip(tx)

    def apply_transaction_to_tip(self, tx):
        if tx.meta.l1_timestamp is None:
            tx.meta.l1_timestamp = self.eth_context.last_l1_timestamp
            tx.meta.l1_block_number = self.l1_block_number
        elif tx.meta.l1_timestamp > self.eth_context.last_l1_timestamp:
            self.eth_context.last_l1_timestamp = tx.meta.l1_timestamp
            self.l1_block_number = tx.meta.l1_block_number
        elif tx.meta.l1_timestamp < self.eth_context.last_l1_timestamp:
            raise Exception("Out of order timestamp!")

        if tx.meta.index is None:
            tx.meta.index = self.rollup_context.current_index + 1
        self.rollup_context.current_index = tx.meta.index
        if tx.meta.queue_index is not None:
            self.current_queue_index = tx.meta.queue_index

        return miner.apply_transaction(tx)

    def apply_batched_transaction(self, tx):
        assert tx.meta.index is not None
        self.apply_indexed_transaction(tx)
        self.rollup_context.current_batch_tx_index = tx.meta.index

    def validate_and_apply_sequencer_transaction(self, tx):
        if tx.meta.queue_origin != QueueOrigin.SEQUENCER:
            raise Exception("Expected sequencer transactions only")
        self.tx_pool.validate_tx(tx)
        return apply_transaction(tx)

    @abc.abstractmethod
    def handle_eth_send_raw_transaction(self, tx):
        return
```

##### `TransactionExecutor.apply_transaction`

The `tx` argument will have an index if it is synced from the canonical transaction chain or if it is synced from a replica.

##### `TransactionExecutor.set_initial_rollup_context`

This function is required for clean startups. It sets the `current_enqueue_index` based on the latest confirmed L1 to L2 transaction.
This allows the sequencer to continue ingesting `QueueOrigin.L1_TO_L2` transactions without skipping or double ingesting a transaction.
It also updates the `current_index` based on the tip of the chain if the indexed number is different. This is used because the
act of adding a block to the chain is asynchronous and sometimes the indices can be off after shutting down. Note that the index is
off by one relative to the block number due to the CTC being zero indexed while the node implementation is not zero indexed.

### Verifier

The `Verifier` is meant to execute transactions so that a fraud prover can observe the computed state
roots and the state roots posted to the State Commitement Chain on Layer One. The fraud prover can
submit a fraud proof when the state roots are observed to be different.

A verifier syncs from a data transport layer that is syncing from L1. It pulls in transactions that have been appended
to the canonical transaction chain and executes them. It does not need to manage timestamps as all transactions in the
canonical transaction chain have a timestamp already. It does not play enqueue transactions.

It syncs based on batches because that eliminates configuration confusion around
what the data transport layer is syncing. The batch indexes will only be present
in the data transport layer if it is syncing from layer one.
`Verifier.sync_transactions_to_tip` cannot give a guarantee on the source of the
transaction as it will return different results based on the configuration of
the data transport layer. It is much safer to sync based on batches to track the
verified transaction index.

```python
class Verifier(TransactionExecutor):
    def __init__(self, options):
        self.type = VERIFIER
        self.backend = L1
        super().set_initial_eth_context()
        super().set_initial_rollup_context()
        self.client = RollupClient(options.l1_transaction_indexer_url)
        self.start()

    def start():
        while True:
            self.sync_tx_batches_to_tip()
            sleep self.poll_interval

    def sync_transactions_to_tip(self):
        latest_indexed_transaction = self.client.get_latest_transaction(self.backend)
        latest_indexed_transaction_index = latest_indexed_transaction.meta.index
        latest_local_transaction_index = self.rollup_context.index

        while latest_indexed_transaction_index != latest_local_transaction_index:
            for tx_index in range(latest_local_transaction_index, latest_indexed_transaction_index + 1):
                tx = self.client.get_transaction(tx_index)
                self.apply_transaction(tx)

            latest_indexed_transaction = self.client.get_latest_transaction()
            latest_indexed_transaction_index = latest_indexed_transaction.index
            latest_local_transaction = self.rollup_context.index

    def sync_tx_batches_to_tip(self):
        latest_indexed_tx_batch = self.client.get_latest_tx_batch()
        latest_indexed_tx_batch_index = tx_batch.batch.index
        local_batch_index = self.rollup_context.current_tx_batch_index

        while latest_indexed_tx_batch_index != local_batch_index:
            for batch_index in range(local_batch_index, latest_indexed_tx_batch_index + 1)
            tx_batch = self.client.get_tx_batch(batch_index)
            for tx in tx_batch.transactions:
                self.apply_batched_transaction(tx)

            latest_indexed_tx_batch = self.client.get_latest_tx_batch()
            latest_indexed_tx_batch_index = tx_batch.batch.index

    def handle_eth_send_raw_transaction(self, tx):
        raise Exception("Cannot accept transactions")
```

### Sequencer

The `Sequencer` holds a privileged role in the system as it is allowed to extend the chain. It holds the keys to the
EOAs in the Layer One AddressManager `OVM_Sequencer` and the `OVM_Proposer`. These roles are the only roles allowed to
extend the Canonical Transaction Chain and the State Commitment Chain. In the future, any account should be able to
call `queueBatchAppend` on the Canonical Transaction Chain to make appending to CTC permissionless.
Note that EOAs must be used to submit batches.

A sequencer syncs from a data transport layer that is syncing from L1. It pulls in enqueue transactions as soon as they are
enqueued and executes them. It must update its timestamps such that it can give a timestamp and blocknumber to queue origin
sequencer transactions that maintains monotonicity.

```python
class Sequencer(Verifier):
    def __init__(self, options):
        self.type = SEQUENCER
        self.backend = L1
        self.client = RollupClient(options.l1_transaction_indexer_url)
        super().set_initial_eth_context()
        super().set_initial_rollup_context()
        super().sync_tx_batches_to_tip()
        self.start()

    def start():
        while True:
            self.sync_queue_to_tip()
            self.sync_tx_batches_to_tip()
            sleep options.poll_interval

    def sync_queue_to_tip(self):
        latest_indexed_queue_element = self.client.get_latest_enqueue()
        latest_indexed_queue_element_index = latest_indexed_queue_element.meta.index
        latest_local_queue_element_index = self.rollup_context.current_queue_index

        while latest_indexed_queue_element_index != latest_local_queue_element:
            for tx_index in range(latest_local_queue_element_index, latest_indexed_queue_element_index):
                tx = self.client.get_queue_element(tx_index)
                self.apply_transaction(tx)

            latest_indexed_queue_element = self.client.get_latest_enqueue()
            latest_indexed_queue_element_index = latest_indexed_queue_element.meta.index
            latest_local_queue_element_index = self.rollup_context.current_queue_index

    def handle_eth_send_raw_transaction(self, tx):
        tx.set_timestamp(self.eth_context.last_l1_timestamp)
        tx.set_blocknumber(self.eth_context.last_l1_blocknumber)
        return validate_and_apply_sequencer_transaction(tx)
```

### Sequencer Replica

A sequencer replica syncs from a data transport layer that is syncing from a sequencer. When it receives a transaction via RPC,
it will ensure that it is at the tip of the replica data transport layer and then switch to syncing from a layer one data transport layer.

```python
class SequencerReplica(Sequencer):
    def __init__(self, options):
        self.type = SEQUENCER
        self.backend = L2
        self.client = RollupClient(options.replica_transaction_indexer_url)
        super().set_initial_eth_context()
        super().set_initial_rollup_context()
        self.start()

   def start():
        while True:
            if self.backend == L2:
                self.sync_transactions_to_tip(self.backend)
            elif self.backend == L1:
                self.sync_queue_to_tip()
                self.sync_tx_batches_to_tip()
            sleep options.poll_interval

    def handle_eth_send_raw_transaction(self, tx):
        if self.backend == L2:
            self.sync_transactions_to_tip(self.backend)
            self.backend = L1
        self.apply_transaction(tx)
```

### `VerifierReplica`

A verifier replica syncs from a data transport layer that is syncing from a sequencer. It will refuse to accept transactions via RPC

```python
class VerifierReplica(Verifier):
    def __init__(self, options):
        self.type = VERIFIER
        self.backend = L2
        self.client = RollupClient(options.replica_transaction_indexer_url)
        super().set_initial_eth_context()
        super().set_initial_rollup_context()
        self.start()

    def start():
        while True:
            self.sync_transactions_to_tip(self.backend)
            sleep self.poll_interval

    def handle_eth_send_raw_transaction(self, tx):
        raise Exception("Cannot accept transactions")
```

## Sequencer Deployment

Deploy:

- Sequencer
- Layer One Data Transport Layer

## Sequencer Upgrades

Deploy new:

- Replica Sequencer
- Replica Data Transport Layer

The replica data transport layer is configured to sync from the sequencer. The replica sequencer is syncing from the replica data transport layer.
The DNS is updated to point `mainnet.optimism.io` to the replica sequencer. When the replica sequencer receives a transaction, it syncs to the
tip of the replica data transport layer, plays the transaction and then switches to syncing from the layer one data transport layer.
