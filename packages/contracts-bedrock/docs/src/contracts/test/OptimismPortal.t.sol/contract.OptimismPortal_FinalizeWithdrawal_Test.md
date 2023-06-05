# OptimismPortal_FinalizeWithdrawal_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismPortal.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)


## State Variables
### _defaultTx

```solidity
Types.WithdrawalTransaction _defaultTx;
```


### _proposedOutputIndex

```solidity
uint256 _proposedOutputIndex;
```


### _proposedBlockNumber

```solidity
uint256 _proposedBlockNumber;
```


### _stateRoot

```solidity
bytes32 _stateRoot;
```


### _storageRoot

```solidity
bytes32 _storageRoot;
```


### _outputRoot

```solidity
bytes32 _outputRoot;
```


### _withdrawalHash

```solidity
bytes32 _withdrawalHash;
```


### _withdrawalProof

```solidity
bytes[] _withdrawalProof;
```


### _outputRootProof

```solidity
Types.OutputRootProof internal _outputRootProof;
```


## Functions
### constructor


```solidity
constructor();
```

### setUp


```solidity
function setUp() public override;
```

### callPortalAndExpectRevert


```solidity
function callPortalAndExpectRevert() external payable;
```

### test_proveWithdrawalTransaction_paused_reverts

Proving withdrawal transactions should revert when paused


```solidity
function test_proveWithdrawalTransaction_paused_reverts() external;
```

### test_proveWithdrawalTransaction_onSelfCall_reverts


```solidity
function test_proveWithdrawalTransaction_onSelfCall_reverts() external;
```

### test_proveWithdrawalTransaction_onInvalidOutputRootProof_reverts


```solidity
function test_proveWithdrawalTransaction_onInvalidOutputRootProof_reverts() external;
```

### test_proveWithdrawalTransaction_onInvalidWithdrawalProof_reverts


```solidity
function test_proveWithdrawalTransaction_onInvalidWithdrawalProof_reverts() external;
```

### test_proveWithdrawalTransaction_replayProve_reverts


```solidity
function test_proveWithdrawalTransaction_replayProve_reverts() external;
```

### test_proveWithdrawalTransaction_replayProveChangedOutputRoot_succeeds


```solidity
function test_proveWithdrawalTransaction_replayProveChangedOutputRoot_succeeds() external;
```

### test_proveWithdrawalTransaction_replayProveChangedOutputRootAndOutputIndex_succeeds


```solidity
function test_proveWithdrawalTransaction_replayProveChangedOutputRootAndOutputIndex_succeeds() external;
```

### test_proveWithdrawalTransaction_validWithdrawalProof_succeeds


```solidity
function test_proveWithdrawalTransaction_validWithdrawalProof_succeeds() external;
```

### test_finalizeWithdrawalTransaction_provenWithdrawalHash_succeeds


```solidity
function test_finalizeWithdrawalTransaction_provenWithdrawalHash_succeeds() external;
```

### test_finalizeWithdrawalTransaction_paused_reverts

Finalizing withdrawal transactions should revert when paused


```solidity
function test_finalizeWithdrawalTransaction_paused_reverts() external;
```

### test_finalizeWithdrawalTransaction_ifWithdrawalNotProven_reverts


```solidity
function test_finalizeWithdrawalTransaction_ifWithdrawalNotProven_reverts() external;
```

### test_finalizeWithdrawalTransaction_ifWithdrawalProofNotOldEnough_reverts


```solidity
function test_finalizeWithdrawalTransaction_ifWithdrawalProofNotOldEnough_reverts() external;
```

### test_finalizeWithdrawalTransaction_timestampLessThanL2OracleStart_reverts


```solidity
function test_finalizeWithdrawalTransaction_timestampLessThanL2OracleStart_reverts() external;
```

### test_finalizeWithdrawalTransaction_ifOutputRootChanges_reverts


```solidity
function test_finalizeWithdrawalTransaction_ifOutputRootChanges_reverts() external;
```

### test_finalizeWithdrawalTransaction_ifOutputTimestampIsNotFinalized_reverts


```solidity
function test_finalizeWithdrawalTransaction_ifOutputTimestampIsNotFinalized_reverts() external;
```

### test_finalizeWithdrawalTransaction_targetFails_fails


```solidity
function test_finalizeWithdrawalTransaction_targetFails_fails() external;
```

### test_finalizeWithdrawalTransaction_onRecentWithdrawal_reverts


```solidity
function test_finalizeWithdrawalTransaction_onRecentWithdrawal_reverts() external;
```

### test_finalizeWithdrawalTransaction_onReplay_reverts


```solidity
function test_finalizeWithdrawalTransaction_onReplay_reverts() external;
```

### test_finalizeWithdrawalTransaction_onInsufficientGas_reverts


```solidity
function test_finalizeWithdrawalTransaction_onInsufficientGas_reverts() external;
```

### test_finalizeWithdrawalTransaction_onReentrancy_reverts


```solidity
function test_finalizeWithdrawalTransaction_onReentrancy_reverts() external;
```

### testDiff_finalizeWithdrawalTransaction_succeeds


```solidity
function testDiff_finalizeWithdrawalTransaction_succeeds(
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

