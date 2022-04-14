# L2OutputOracle



> L2OutputOracle





## Methods

### HISTORICAL_TOTAL_BLOCKS

```solidity
function HISTORICAL_TOTAL_BLOCKS() external view returns (uint256)
```

The number of blocks in the chain before the first block in this contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### L2_BLOCK_TIME

```solidity
function L2_BLOCK_TIME() external view returns (uint256)
```

The time between blocks on L2.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### STARTING_BLOCK_TIMESTAMP

```solidity
function STARTING_BLOCK_TIMESTAMP() external view returns (uint256)
```

The timestamp of the first L2 block recorded in this contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### SUBMISSION_INTERVAL

```solidity
function SUBMISSION_INTERVAL() external view returns (uint256)
```

The interval in seconds at which checkpoints must be submitted.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### appendL2Output

```solidity
function appendL2Output(bytes32 _l2Output, uint256 _l2timestamp, bytes32 _l1Blockhash, uint256 _l1Blocknumber) external payable
```

Accepts an L2 outputRoot and the timestamp of the corresponding L2 block. The timestamp must be equal to the current value returned by `nextTimestamp()` in order to be accepted. This function may only be called by the Sequencer.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Output | bytes32 | The L2 output of the checkpoint block.
| _l2timestamp | uint256 | The L2 block timestamp that resulted in _l2Output.
| _l1Blockhash | bytes32 | A block hash which must be included in the current chain.
| _l1Blocknumber | uint256 | The block number with the specified block hash.

### computeL2BlockNumber

```solidity
function computeL2BlockNumber(uint256 _l2timestamp) external view returns (uint256)
```

Computes the L2 block number given a target L2 block timestamp.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2timestamp | uint256 | The L2 block timestamp of the target block.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### getL2Output

```solidity
function getL2Output(uint256 _l2Timestamp) external view returns (bytes32)
```

Returns the L2 output root given a target L2 block timestamp. Returns 0 if none is found.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Timestamp | uint256 | The L2 block timestamp of the target block.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

### latestBlockTimestamp

```solidity
function latestBlockTimestamp() external view returns (uint256)
```

The timestamp of the most recent L2 block recorded in this contract.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### nextTimestamp

```solidity
function nextTimestamp() external view returns (uint256)
```

Computes the timestamp of the next L2 block that needs to be checkpointed.




#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined



## Events

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### l2OutputAppended

```solidity
event l2OutputAppended(bytes32 indexed _l2Output, uint256 indexed _l2timestamp)
```

Emitted when an output is appended.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _l2Output `indexed` | bytes32 | undefined |
| _l2timestamp `indexed` | uint256 | undefined |



