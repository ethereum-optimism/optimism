# ResourceMetering_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/ResourceMetering.t.sol)

**Inherits:**
Test

The tests are based on the default config values. It is expected that
the config values used in these tests are ran in production.


## State Variables
### meter

```solidity
MeterUser internal meter;
```


### initialBlockNum

```solidity
uint64 initialBlockNum;
```


## Functions
### setUp


```solidity
function setUp() public;
```

### test_meter_initialResourceParams_succeeds


```solidity
function test_meter_initialResourceParams_succeeds() external;
```

### test_meter_updateParamsNoChange_succeeds


```solidity
function test_meter_updateParamsNoChange_succeeds() external;
```

### test_meter_updateOneEmptyBlock_succeeds


```solidity
function test_meter_updateOneEmptyBlock_succeeds() external;
```

### test_meter_updateTwoEmptyBlocks_succeeds


```solidity
function test_meter_updateTwoEmptyBlocks_succeeds() external;
```

### test_meter_updateTenEmptyBlocks_succeeds


```solidity
function test_meter_updateTenEmptyBlocks_succeeds() external;
```

### test_meter_updateNoGasDelta_succeeds


```solidity
function test_meter_updateNoGasDelta_succeeds() external;
```

### test_meter_useMax_succeeds


```solidity
function test_meter_useMax_succeeds() external;
```

### test_meter_denominatorEq1_reverts

This tests that the metered modifier reverts if
the ResourceConfig baseFeeMaxChangeDenominator
is set to 1.
Since the metered modifier internally calls
solmate's powWad function, it will revert
with the error string "UNDEFINED" since the
first parameter will be computed as 0.


```solidity
function test_meter_denominatorEq1_reverts() external;
```

### test_meter_useMoreThanMax_reverts


```solidity
function test_meter_useMoreThanMax_reverts() external;
```

### testFuzz_meter_largeBlockDiff_succeeds


```solidity
function testFuzz_meter_largeBlockDiff_succeeds(uint64 _amount, uint256 _blockDiff) external;
```

