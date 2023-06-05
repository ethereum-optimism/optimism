# L2OutputOracleTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2OutputOracle.t.sol)

**Inherits:**
[L2OutputOracle_Initializer](/contracts/test/CommonTest.t.sol/contract.L2OutputOracle_Initializer.md)


## State Variables
### proposedOutput1

```solidity
bytes32 proposedOutput1 = keccak256(abi.encode(1));
```


## Functions
### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_constructor_badTimestamp_reverts


```solidity
function test_constructor_badTimestamp_reverts() external;
```

### test_constructor_l2BlockTimeZero_reverts


```solidity
function test_constructor_l2BlockTimeZero_reverts() external;
```

### test_constructor_submissionInterval_reverts


```solidity
function test_constructor_submissionInterval_reverts() external;
```

### test_latestBlockNumber_succeeds

Getter Tests *


```solidity
function test_latestBlockNumber_succeeds() external;
```

### test_getL2Output_succeeds


```solidity
function test_getL2Output_succeeds() external;
```

### test_getL2OutputIndexAfter_sameBlock_succeeds


```solidity
function test_getL2OutputIndexAfter_sameBlock_succeeds() external;
```

### test_getL2OutputIndexAfter_previousBlock_succeeds


```solidity
function test_getL2OutputIndexAfter_previousBlock_succeeds() external;
```

### test_getL2OutputIndexAfter_multipleOutputsExist_succeeds


```solidity
function test_getL2OutputIndexAfter_multipleOutputsExist_succeeds() external;
```

### test_getL2OutputIndexAfter_noOutputsExis_reverts


```solidity
function test_getL2OutputIndexAfter_noOutputsExis_reverts() external;
```

### test_nextBlockNumber_succeeds


```solidity
function test_nextBlockNumber_succeeds() external;
```

### test_computeL2Timestamp_succeeds


```solidity
function test_computeL2Timestamp_succeeds() external;
```

### test_proposeL2Output_proposeAnotherOutput_succeeds

Propose Tests - Happy Path *


```solidity
function test_proposeL2Output_proposeAnotherOutput_succeeds() public;
```

### test_proposeWithBlockhashAndHeight_succeeds


```solidity
function test_proposeWithBlockhashAndHeight_succeeds() external;
```

### test_proposeL2Output_notProposer_reverts

Propose Tests - Sad Path *


```solidity
function test_proposeL2Output_notProposer_reverts() external;
```

### test_proposeL2Output_emptyOutput_reverts


```solidity
function test_proposeL2Output_emptyOutput_reverts() external;
```

### test_proposeL2Output_unexpectedBlockNumber_reverts


```solidity
function test_proposeL2Output_unexpectedBlockNumber_reverts() external;
```

### test_proposeL2Output_futureTimetamp_reverts


```solidity
function test_proposeL2Output_futureTimetamp_reverts() external;
```

### test_proposeL2Output_wrongFork_reverts


```solidity
function test_proposeL2Output_wrongFork_reverts() external;
```

### test_proposeL2Output_unmatchedBlockhash_reverts


```solidity
function test_proposeL2Output_unmatchedBlockhash_reverts() external;
```

### test_deleteOutputs_singleOutput_succeeds

Delete Tests - Happy Path *


```solidity
function test_deleteOutputs_singleOutput_succeeds() external;
```

### test_deleteOutputs_multipleOutputs_succeeds


```solidity
function test_deleteOutputs_multipleOutputs_succeeds() external;
```

### test_deleteL2Outputs_ifNotChallenger_reverts

Delete Tests - Sad Path *


```solidity
function test_deleteL2Outputs_ifNotChallenger_reverts() external;
```

### test_deleteL2Outputs_nonExistent_reverts


```solidity
function test_deleteL2Outputs_nonExistent_reverts() external;
```

### test_deleteL2Outputs_afterLatest_reverts


```solidity
function test_deleteL2Outputs_afterLatest_reverts() external;
```

### test_deleteL2Outputs_finalized_reverts


```solidity
function test_deleteL2Outputs_finalized_reverts() external;
```

