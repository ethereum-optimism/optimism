# Proxy_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Proxy.t.sol)

**Inherits:**
Test


## State Variables
### alice

```solidity
address alice = address(64);
```


### IMPLEMENTATION_KEY

```solidity
bytes32 internal constant IMPLEMENTATION_KEY = bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1);
```


### OWNER_KEY

```solidity
bytes32 internal constant OWNER_KEY = bytes32(uint256(keccak256("eip1967.proxy.admin")) - 1);
```


### proxy

```solidity
Proxy proxy;
```


### simpleStorage

```solidity
SimpleStorage simpleStorage;
```


## Functions
### setUp


```solidity
function setUp() external;
```

### test_implementationKey_succeeds


```solidity
function test_implementationKey_succeeds() external;
```

### test_ownerKey_succeeds


```solidity
function test_ownerKey_succeeds() external;
```

### test_proxyCallToImp_notAdmin_succeeds


```solidity
function test_proxyCallToImp_notAdmin_succeeds() external;
```

### test_ownerProxyCall_notAdmin_succeeds


```solidity
function test_ownerProxyCall_notAdmin_succeeds() external;
```

### test_delegatesToImpl_succeeds


```solidity
function test_delegatesToImpl_succeeds() external;
```

### test_upgradeToAndCall_succeeds


```solidity
function test_upgradeToAndCall_succeeds() external;
```

### test_upgradeToAndCall_functionDoesNotExist_reverts


```solidity
function test_upgradeToAndCall_functionDoesNotExist_reverts() external;
```

### test_upgradeToAndCall_isPayable_succeeds


```solidity
function test_upgradeToAndCall_isPayable_succeeds() external;
```

### test_upgradeTo_clashingFunctionSignatures_succeeds


```solidity
function test_upgradeTo_clashingFunctionSignatures_succeeds() external;
```

### test_implementation_zeroAddressCaller_succeeds


```solidity
function test_implementation_zeroAddressCaller_succeeds() external;
```

### test_implementation_isZeroAddress_reverts


```solidity
function test_implementation_isZeroAddress_reverts() external;
```

## Events
### Upgraded

```solidity
event Upgraded(address indexed implementation);
```

### AdminChanged

```solidity
event AdminChanged(address previousAdmin, address newAdmin);
```

