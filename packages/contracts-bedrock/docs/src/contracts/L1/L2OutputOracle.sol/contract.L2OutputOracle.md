# L2OutputOracle
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L1/L2OutputOracle.sol)

**Inherits:**
Initializable, [Semver](/contracts/universal/Semver.sol/contract.Semver.md)

The L2OutputOracle contains an array of L2 state outputs, where each output is a
commitment to the state of the L2 chain. Other contracts like the OptimismPortal use
these outputs to verify information about the state of L2.


## State Variables
### SUBMISSION_INTERVAL
The interval in L2 blocks at which checkpoints must be submitted. Although this is
immutable, it can safely be modified by upgrading the implementation contract.


```solidity
uint256 public immutable SUBMISSION_INTERVAL;
```


### L2_BLOCK_TIME
The time between L2 blocks in seconds. Once set, this value MUST NOT be modified.


```solidity
uint256 public immutable L2_BLOCK_TIME;
```


### CHALLENGER
The address of the challenger. Can be updated via upgrade.


```solidity
address public immutable CHALLENGER;
```


### PROPOSER
The address of the proposer. Can be updated via upgrade.


```solidity
address public immutable PROPOSER;
```


### FINALIZATION_PERIOD_SECONDS
Minimum time (in seconds) that must elapse before a withdrawal can be finalized.


```solidity
uint256 public immutable FINALIZATION_PERIOD_SECONDS;
```


### startingBlockNumber
The number of the first L2 block recorded in this contract.


```solidity
uint256 public startingBlockNumber;
```


### startingTimestamp
The timestamp of the first L2 block recorded in this contract.


```solidity
uint256 public startingTimestamp;
```


### l2Outputs
Array of L2 output proposals.


```solidity
Types.OutputProposal[] internal l2Outputs;
```


## Functions
### constructor


```solidity
constructor(
    uint256 _submissionInterval,
    uint256 _l2BlockTime,
    uint256 _startingBlockNumber,
    uint256 _startingTimestamp,
    address _proposer,
    address _challenger,
    uint256 _finalizationPeriodSeconds
) Semver(1, 3, 0);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_submissionInterval`|`uint256`| Interval in blocks at which checkpoints must be submitted.|
|`_l2BlockTime`|`uint256`|        The time per L2 block, in seconds.|
|`_startingBlockNumber`|`uint256`|The number of the first L2 block.|
|`_startingTimestamp`|`uint256`|  The timestamp of the first L2 block.|
|`_proposer`|`address`|           The address of the proposer.|
|`_challenger`|`address`|         The address of the challenger.|
|`_finalizationPeriodSeconds`|`uint256`||


### initialize

Initializer.


```solidity
function initialize(uint256 _startingBlockNumber, uint256 _startingTimestamp) public initializer;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_startingBlockNumber`|`uint256`|Block number for the first recoded L2 block.|
|`_startingTimestamp`|`uint256`|  Timestamp for the first recoded L2 block.|


### deleteL2Outputs

Deletes all output proposals after and including the proposal that corresponds to
the given output index. Only the challenger address can delete outputs.


```solidity
function deleteL2Outputs(uint256 _l2OutputIndex) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2OutputIndex`|`uint256`|Index of the first L2 output to be deleted. All outputs after this output will also be deleted.|


### proposeL2Output

Accepts an outputRoot and the timestamp of the corresponding L2 block. The timestamp
must be equal to the current value returned by `nextTimestamp()` in order to be
accepted. This function may only be called by the Proposer.


```solidity
function proposeL2Output(bytes32 _outputRoot, uint256 _l2BlockNumber, bytes32 _l1BlockHash, uint256 _l1BlockNumber)
    external
    payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_outputRoot`|`bytes32`|   The L2 output of the checkpoint block.|
|`_l2BlockNumber`|`uint256`|The L2 block number that resulted in _outputRoot.|
|`_l1BlockHash`|`bytes32`|  A block hash which must be included in the current chain.|
|`_l1BlockNumber`|`uint256`|The block number with the specified block hash.|


### getL2Output

Returns an output by index. Exists because Solidity's array access will return a
tuple instead of a struct.


```solidity
function getL2Output(uint256 _l2OutputIndex) external view returns (Types.OutputProposal memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2OutputIndex`|`uint256`|Index of the output to return.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`OutputProposal.Types`|The output at the given index.|


### getL2OutputIndexAfter

Returns the index of the L2 output that checkpoints a given L2 block number. Uses a
binary search to find the first output greater than or equal to the given block.


```solidity
function getL2OutputIndexAfter(uint256 _l2BlockNumber) public view returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2BlockNumber`|`uint256`|L2 block number to find a checkpoint for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Index of the first checkpoint that commits to the given L2 block number.|


### getL2OutputAfter

Returns the L2 output proposal that checkpoints a given L2 block number. Uses a
binary search to find the first output greater than or equal to the given block.


```solidity
function getL2OutputAfter(uint256 _l2BlockNumber) external view returns (Types.OutputProposal memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2BlockNumber`|`uint256`|L2 block number to find a checkpoint for.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`OutputProposal.Types`|First checkpoint that commits to the given L2 block number.|


### latestOutputIndex

Returns the number of outputs that have been proposed. Will revert if no outputs
have been proposed yet.


```solidity
function latestOutputIndex() external view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|The number of outputs that have been proposed.|


### nextOutputIndex

Returns the index of the next output to be proposed.


```solidity
function nextOutputIndex() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|The index of the next output to be proposed.|


### latestBlockNumber

Returns the block number of the latest submitted L2 output proposal. If no proposals
been submitted yet then this function will return the starting block number.


```solidity
function latestBlockNumber() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Latest submitted L2 block number.|


### nextBlockNumber

Computes the block number of the next L2 block that needs to be checkpointed.


```solidity
function nextBlockNumber() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Next L2 block number.|


### computeL2Timestamp

Returns the L2 timestamp corresponding to a given L2 block number.


```solidity
function computeL2Timestamp(uint256 _l2BlockNumber) public view returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l2BlockNumber`|`uint256`|The L2 block number of the target block.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|L2 timestamp of the given block.|


## Events
### OutputProposed
Emitted when an output is proposed.


```solidity
event OutputProposed(
    bytes32 indexed outputRoot, uint256 indexed l2OutputIndex, uint256 indexed l2BlockNumber, uint256 l1Timestamp
);
```

### OutputsDeleted
Emitted when outputs are deleted.


```solidity
event OutputsDeleted(uint256 indexed prevNextOutputIndex, uint256 indexed newNextOutputIndex);
```

