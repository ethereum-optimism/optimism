# GasPriceOracle_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/GasPriceOracle.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## State Variables
### gasOracle

```solidity
GasPriceOracle gasOracle;
```


### l1Block

```solidity
L1Block l1Block;
```


### depositor

```solidity
address depositor;
```


### number

```solidity
uint64 constant number = 10;
```


### timestamp

```solidity
uint64 constant timestamp = 11;
```


### basefee

```solidity
uint256 constant basefee = 100;
```


### hash

```solidity
bytes32 constant hash = bytes32(uint256(64));
```


### sequenceNumber

```solidity
uint64 constant sequenceNumber = 0;
```


### batcherHash

```solidity
bytes32 constant batcherHash = bytes32(uint256(777));
```


### l1FeeOverhead

```solidity
uint256 constant l1FeeOverhead = 310;
```


### l1FeeScalar

```solidity
uint256 constant l1FeeScalar = 10;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

### test_l1BaseFee_succeeds


```solidity
function test_l1BaseFee_succeeds() external;
```

### test_gasPrice_succeeds


```solidity
function test_gasPrice_succeeds() external;
```

### test_baseFee_succeeds


```solidity
function test_baseFee_succeeds() external;
```

### test_scalar_succeeds


```solidity
function test_scalar_succeeds() external;
```

### test_overhead_succeeds


```solidity
function test_overhead_succeeds() external;
```

### test_decimals_succeeds


```solidity
function test_decimals_succeeds() external;
```

### test_setGasPrice_doesNotExist_reverts


```solidity
function test_setGasPrice_doesNotExist_reverts() external;
```

### test_setL1BaseFee_doesNotExist_reverts


```solidity
function test_setL1BaseFee_doesNotExist_reverts() external;
```

## Events
### OverheadUpdated

```solidity
event OverheadUpdated(uint256);
```

### ScalarUpdated

```solidity
event ScalarUpdated(uint256);
```

### DecimalsUpdated

```solidity
event DecimalsUpdated(uint256);
```

