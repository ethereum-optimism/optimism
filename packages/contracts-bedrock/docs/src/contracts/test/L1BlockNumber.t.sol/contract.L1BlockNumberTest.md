# L1BlockNumberTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L1BlockNumber.t.sol)

**Inherits:**
Test


## State Variables
### lb

```solidity
L1Block lb;
```


### bn

```solidity
L1BlockNumber bn;
```


### number

```solidity
uint64 constant number = 99;
```


## Functions
### setUp


```solidity
function setUp() external;
```

### test_getL1BlockNumber_succeeds


```solidity
function test_getL1BlockNumber_succeeds() external;
```

### test_fallback_succeeds


```solidity
function test_fallback_succeeds() external;
```

### test_receive_succeeds


```solidity
function test_receive_succeeds() external;
```

