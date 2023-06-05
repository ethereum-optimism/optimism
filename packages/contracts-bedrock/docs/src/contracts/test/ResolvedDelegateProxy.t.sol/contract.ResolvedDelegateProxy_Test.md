# ResolvedDelegateProxy_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ResolvedDelegateProxy.t.sol)

**Inherits:**
Test


## State Variables
### addressManager

```solidity
AddressManager internal addressManager;
```


### impl

```solidity
SimpleImplementation internal impl;
```


### proxy

```solidity
SimpleImplementation internal proxy;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### testFuzz_fallback_delegateCallFoo_succeeds


```solidity
function testFuzz_fallback_delegateCallFoo_succeeds(uint256 x) public;
```

### test_fallback_delegateCallBar_reverts


```solidity
function test_fallback_delegateCallBar_reverts() public;
```

### test_fallback_addressManagerNotSet_reverts


```solidity
function test_fallback_addressManagerNotSet_reverts() public;
```

