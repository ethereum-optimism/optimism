# Bytes_equal_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Bytes.t.sol)

**Inherits:**
Test


## Functions
### manualEq

Manually checks equality of two dynamic `bytes` arrays in memory.


```solidity
function manualEq(bytes memory _a, bytes memory _b) internal pure returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_a`|`bytes`|The first `bytes` array to compare.|
|`_b`|`bytes`|The second `bytes` array to compare.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|True if the two `bytes` arrays are equal in memory.|


### testFuzz_equal_notEqual_works

Tests that the `equal` function in the `Bytes` library returns `false` if given two
non-equal byte arrays.


```solidity
function testFuzz_equal_notEqual_works(bytes memory _a, bytes memory _b) public;
```

### testDiff_equal_works

Test whether or not the `equal` function in the `Bytes` library is equivalent to
manually checking equality of the two dynamic `bytes` arrays in memory.


```solidity
function testDiff_equal_works(bytes memory _a, bytes memory _b) public;
```

