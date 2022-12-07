# StateCommitmentChain



> StateCommitmentChain



*The State Commitment Chain (SCC) contract contains a list of proposed state roots which Proposers assert to be a result of each transaction in the Canonical Transaction Chain (CTC). Elements here have a 1:1 correspondence with transactions in the CTC, and should be the unique state root calculated off-chain by applying the canonical transactions one by one.*

## Methods

### FRAUD_PROOF_WINDOW

```solidity
function FRAUD_PROOF_WINDOW() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### SEQUENCER_PUBLISH_WINDOW

```solidity
function SEQUENCER_PUBLISH_WINDOW() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### appendStateBatch

```solidity
function appendStateBatch(bytes32[] _batch, uint256 _shouldStartAtElement) external nonpayable
```

Appends a batch of state roots to the chain.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batch | bytes32[] | Batch of state roots.
| _shouldStartAtElement | uint256 | Index of the element at which this batch should start.

### batches

```solidity
function batches() external view returns (contract IChainStorageContainer)
```

Accesses the batch storage container.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract IChainStorageContainer | Reference to the batch storage container.

### deleteStateBatch

```solidity
function deleteStateBatch(Lib_OVMCodec.ChainBatchHeader _batchHeader) external nonpayable
```

Deletes all state roots after (and including) a given batch.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | Lib_OVMCodec.ChainBatchHeader | Header of the batch to start deleting from.

### getLastSequencerTimestamp

```solidity
function getLastSequencerTimestamp() external view returns (uint256 _lastSequencerTimestamp)
```

Retrieves the timestamp of the last batch submitted by the sequencer.




#### Returns

| Name | Type | Description |
|---|---|---|
| _lastSequencerTimestamp | uint256 | Last sequencer batch timestamp.

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

### insideFraudProofWindow

```solidity
function insideFraudProofWindow(Lib_OVMCodec.ChainBatchHeader _batchHeader) external view returns (bool _inside)
```

Checks whether a given batch is still inside its fraud proof window.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchHeader | Lib_OVMCodec.ChainBatchHeader | Header of the batch to check.

#### Returns

| Name | Type | Description |
|---|---|---|
| _inside | bool | Whether or not the batch is inside the fraud proof window.

### libAddressManager

```solidity
function libAddressManager() external view returns (contract Lib_AddressManager)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract Lib_AddressManager | undefined

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

### verifyStateCommitment

```solidity
function verifyStateCommitment(bytes32 _element, Lib_OVMCodec.ChainBatchHeader _batchHeader, Lib_OVMCodec.ChainInclusionProof _proof) external view returns (bool)
```

Verifies a batch inclusion proof.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _element | bytes32 | Hash of the element to verify a proof for.
| _batchHeader | Lib_OVMCodec.ChainBatchHeader | Header of the batch in which the element was included.
| _proof | Lib_OVMCodec.ChainInclusionProof | Merkle inclusion proof for the element.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | Whether or not the batch inclusion proof is verified.



## Events

### StateBatchAppended

```solidity
event StateBatchAppended(uint256 indexed _batchIndex, bytes32 _batchRoot, uint256 _batchSize, uint256 _prevTotalElements, bytes _extraData)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchIndex `indexed` | uint256 | undefined |
| _batchRoot  | bytes32 | undefined |
| _batchSize  | uint256 | undefined |
| _prevTotalElements  | uint256 | undefined |
| _extraData  | bytes | undefined |

### StateBatchDeleted

```solidity
event StateBatchDeleted(uint256 indexed _batchIndex, bytes32 _batchRoot)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _batchIndex `indexed` | uint256 | undefined |
| _batchRoot  | bytes32 | undefined |



