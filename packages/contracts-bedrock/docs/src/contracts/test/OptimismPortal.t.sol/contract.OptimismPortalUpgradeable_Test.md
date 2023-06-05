# OptimismPortalUpgradeable_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/OptimismPortal.t.sol)

**Inherits:**
[Portal_Initializer](/contracts/test/CommonTest.t.sol/contract.Portal_Initializer.md)


## State Variables
### proxy

```solidity
Proxy internal proxy;
```


### initialBlockNum

```solidity
uint64 initialBlockNum;
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_params_initValuesOnProxy_succeeds


```solidity
function test_params_initValuesOnProxy_succeeds() external;
```

### test_initialize_cannotInitProxy_reverts


```solidity
function test_initialize_cannotInitProxy_reverts() external;
```

### test_initialize_cannotInitImpl_reverts


```solidity
function test_initialize_cannotInitImpl_reverts() external;
```

### test_upgradeToAndCall_upgrading_succeeds


```solidity
function test_upgradeToAndCall_upgrading_succeeds() external;
```

