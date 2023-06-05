# OptimismPortal_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismPortal.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)


## Functions
### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_pause_succeeds

The OptimismPortal can be paused by the GUARDIAN


```solidity
function test_pause_succeeds() external;
```

### test_pause_onlyGuardian_reverts

The OptimismPortal reverts when an account that is not the
GUARDIAN calls `pause()`


```solidity
function test_pause_onlyGuardian_reverts() external;
```

### test_unpause_succeeds

The OptimismPortal can be unpaused by the GUARDIAN


```solidity
function test_unpause_succeeds() external;
```

### test_unpause_onlyGuardian_reverts

The OptimismPortal reverts when an account that is not
the GUARDIAN calls `unpause()`


```solidity
function test_unpause_onlyGuardian_reverts() external;
```

### test_receive_succeeds


```solidity
function test_receive_succeeds() external;
```

### test_depositTransaction_contractCreation_reverts


```solidity
function test_depositTransaction_contractCreation_reverts() external;
```

### test_depositTransaction_largeData_reverts

Prevent deposits from being too large to have a sane upper bound
on unsafe blocks sent over the p2p network.


```solidity
function test_depositTransaction_largeData_reverts() external;
```

### test_depositTransaction_smallGasLimit_reverts

Prevent gasless deposits from being force processed in L2 by
ensuring that they have a large enough gas limit set.


```solidity
function test_depositTransaction_smallGasLimit_reverts() external;
```

### testFuzz_depositTransaction_smallGasLimit_succeeds

Fuzz for too small of gas limits


```solidity
function testFuzz_depositTransaction_smallGasLimit_succeeds(bytes memory _data, bool _shouldFail) external;
```

### test_minimumGasLimit_succeeds

Ensure that the 0 calldata case is covered and there is a linearly
increasing gas limit for larger calldata sizes.


```solidity
function test_minimumGasLimit_succeeds() external;
```

### test_depositTransaction_noValueEOA_succeeds


```solidity
function test_depositTransaction_noValueEOA_succeeds() external;
```

### test_depositTransaction_noValueContract_succeeds


```solidity
function test_depositTransaction_noValueContract_succeeds() external;
```

### test_depositTransaction_createWithZeroValueForEOA_succeeds


```solidity
function test_depositTransaction_createWithZeroValueForEOA_succeeds() external;
```

### test_depositTransaction_createWithZeroValueForContract_succeeds


```solidity
function test_depositTransaction_createWithZeroValueForContract_succeeds() external;
```

### test_depositTransaction_withEthValueFromEOA_succeeds


```solidity
function test_depositTransaction_withEthValueFromEOA_succeeds() external;
```

### test_depositTransaction_withEthValueFromContract_succeeds


```solidity
function test_depositTransaction_withEthValueFromContract_succeeds() external;
```

### test_depositTransaction_withEthValueAndEOAContractCreation_succeeds


```solidity
function test_depositTransaction_withEthValueAndEOAContractCreation_succeeds() external;
```

### test_depositTransaction_withEthValueAndContractContractCreation_succeeds


```solidity
function test_depositTransaction_withEthValueAndContractContractCreation_succeeds() external;
```

### test_simple_isOutputFinalized_succeeds


```solidity
function test_simple_isOutputFinalized_succeeds() external;
```

### test_isOutputFinalized_succeeds


```solidity
function test_isOutputFinalized_succeeds() external;
```

## Events
### Paused

```solidity
event Paused(address);
```

### Unpaused

```solidity
event Unpaused(address);
```

