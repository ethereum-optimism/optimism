# L2OutputOracleUpgradeable_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2OutputOracle.t.sol)

**Inherits:**
[L2OutputOracle_Initializer](/contracts/test/CommonTest.t.sol/contract.L2OutputOracle_Initializer.md)


## State Variables
### proxy

```solidity
Proxy internal proxy;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_initValuesOnProxy_succeeds


```solidity
function test_initValuesOnProxy_succeeds() external;
```

### test_initializeProxy_alreadyInitialized_reverts


```solidity
function test_initializeProxy_alreadyInitialized_reverts() external;
```

### test_initializeImpl_alreadyInitialized_reverts


```solidity
function test_initializeImpl_alreadyInitialized_reverts() external;
```

### test_upgrading_succeeds


```solidity
function test_upgrading_succeeds() external;
```

