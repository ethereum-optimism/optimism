# CanonicalTransactionChain



> CanonicalTransactionChain



*The Canonical Transaction Chain (CTC) contract is an append-only log of transactions which must be applied to the rollup state. It defines the ordering of rollup transactions by writing them to the &#39;CTC:batches&#39; instance of the Chain Storage Container. The CTC also allows any account to &#39;enqueue&#39; an L2 transaction, which will require that the Sequencer will eventually append it to the rollup state.*

## Methods

### MAX_ROLLUP_TX_SIZE

```solidity
function MAX_ROLLUP_TX_SIZE() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### MIN_ROLLUP_TX_GAS

```solidity
function MIN_ROLLUP_TX_GAS() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### appendSequencerBatch

```solidity
function appendSequencerBatch() external nonpayable
```

Allows the sequencer to append a batch of transactions.

*This function uses a custom encoding scheme for efficiency reasons. .param _shouldStartAtElement Specific batch we expect to start appending to. .param _totalElementsToAppend Total number of batch elements we expect to append. .param _contexts Array of batch contexts. .param _transactionDataFields Array of raw transaction data.*


### batches

```solidity
function batches() external view returns (contract IChainStorageContainer)
```

Accesses the batch storage container.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract IChainStorageContainer | Reference to the batch storage container.

### enqueue

```solidity
function enqueue(address _target, uint256 _gasLimit, bytes _data) external nonpayable
```

Adds a transaction to the queue.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _target | address | Target L2 contract to send the transaction to.
| _gasLimit | uint256 | Gas limit for the enqueued L2 transaction.
| _data | bytes | Transaction data.

### enqueueGasCost

```solidity
function enqueueGasCost() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### enqueueL2GasPrepaid

```solidity
function enqueueL2GasPrepaid() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### getLastBlockNumber

```solidity
function getLastBlockNumber() external view returns (uint40)
```

Returns the blocknumber of the last transaction.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint40 | Blocknumber for the last transaction.

### getLastTimestamp

```solidity
function getLastTimestamp() external view returns (uint40)
```

Returns the timestamp of the last transaction.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint40 | Timestamp for the last transaction.

### getNextQueueIndex

```solidity
function getNextQueueIndex() external view returns (uint40)
```

Returns the index of the next element to be enqueued.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint40 | Index for the next queue element.

### getNumPendingQueueElements

```solidity
function getNumPendingQueueElements() external view returns (uint40)
```

Get the number of queue elements which have not yet been included.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint40 | Number of pending queue elements.

### getQueueElement

```solidity
function getQueueElement(uint256 _index) external view returns (struct Lib_OVMCodec.QueueElement _element)
```

Gets the queue element at a particular index.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _index | uint256 | Index of the queue element to access.

#### Returns

| Name | Type | Description |
|---|---|---|
| _element | Lib_OVMCodec.QueueElement | Queue element at the given index.

### getQueueLength

```solidity
function getQueueLength() external view returns (uint40)
```

Retrieves the length of the queue, including both pending and canonical transactions.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint40 | Length of the queue.

### getTotalBatches

```solidity
function getTotalBatches() external view returns (uint256 _totalBatches)
```

Retrieves the total number of batches submitted.




#### Returns

| Name | Type | Description |
|---|---|---|
| _totalBatches | uint256 | Total submitted batches.

### getTotalElements

```solidity
function getTotalElements() external view returns (uint256 _totalElements)
```

Retrieves the total number of elements submitted.




#### Returns

| Name | Type | Description |
|---|---|---|
| _totalElements | uint256 | Total submitted elements.

### l2GasDiscountDivisor

```solidity
function l2GasDiscountDivisor() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### libAddressManager

```solidity
function libAddressManager() external view returns (contract Lib_AddressManager)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract Lib_AddressManager | undefined

### maxTransactionGasLimit

```solidity
function maxTransactionGasLimit() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### resolve

```solidity
function resolve(string _name) external view returns (address)
```

Resolves the address associated with a given name.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _name | string | Name to resolve an address for.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address associated with the given name.

### setGasParams

```solidity
function setGasParams(uint256 _l2GasDiscountDivisor, uint256 _enqueueGasCost) external nonpayable
```

Allows the Burn Admin to update the parameters which determine the amount of gas to burn. The value of enqueueL2GasPrepaid is immediately updated as well.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2GasDiscountDivisor | uint256 | The ratio of the cost of L1 gas to the cost of L2 gas
| _enqueueGasCost | uint256 | The approximate cost of calling the enqueue function



## Events

### L2GasParamsUpdated

```solidity
event L2GasParamsUpdated(uint256 l2GasDiscountDivisor, uint256 enqueueGasCost, uint256 enqueueL2GasPrepaid)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| l2GasDiscountDivisor  | uint256 | undefined |
| enqueueGasCost  | uint256 | undefined |
| enqueueL2GasPrepaid  | uint256 | undefined |

### QueueBatchAppended

```solidity
event QueueBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _startingQueueIndex  | uint256 | undefined |
| _numQueueElements  | uint256 | undefined |
| _totalElements  | uint256 | undefined |

### SequencerBatchAppended

```solidity
event SequencerBatchAppended(uint256 _startingQueueIndex, uint256 _numQueueElements, uint256 _totalElements)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _startingQueueIndex  | uint256 | undefined |
| _numQueueElements  | uint256 | undefined |
| _totalElements  | uint256 | undefined |

### TransactionBatchAppended

```solidity
event TransactionBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchIndex `indexed` | uint256 | undefined |
| _batchRoot  | bytes32 | undefined |
| _batchSize  | uint256 | undefined |
| _prevTotalElements  | uint256 | undefined |
| _extraData  | bytes | undefined |

### TransactionEnqueued

```solidity
event TransactionEnqueued(address indexed _l1TxOrigin, address indexed _target, uint256 _gasLimit, bytes _data, uint256 indexed _queueIndex, uint256 _timestamp)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _l1TxOrigin `indexed` | address | undefined |
| _target `indexed` | address | undefined |
| _gasLimit  | uint256 | undefined |
| _data  | bytes | undefined |
| _queueIndex `indexed` | uint256 | undefined |
| _timestamp  | uint256 | undefined |



