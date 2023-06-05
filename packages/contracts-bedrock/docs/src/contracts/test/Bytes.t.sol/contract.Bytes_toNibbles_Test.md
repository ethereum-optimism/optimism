# Bytes_toNibbles_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Bytes.t.sol)

**Inherits:**
Test


## Functions
### _toNibblesYul

Diffs the test Solidity version of `toNibbles` against the Yul version.


```solidity
function _toNibblesYul(bytes memory _bytes) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bytes`|`bytes`|The `bytes` array to convert to nibbles.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Yul version of `toNibbles` applied to `_bytes`.|


### test_toNibbles_expectedResult5Bytes_works

Tests that, given an input of 5 bytes, the `toNibbles` function returns an array of
10 nibbles corresponding to the input data.


```solidity
function test_toNibbles_expectedResult5Bytes_works() public;
```

### test_toNibbles_expectedResult128Bytes_works

Tests that, given an input of 128 bytes, the `toNibbles` function returns an array
of 256 nibbles corresponding to the input data. This test exists to ensure that,
given a large input, the `toNibbles` function works as expected.


```solidity
function test_toNibbles_expectedResult128Bytes_works() public;
```

### test_toNibbles_zeroLengthInput_works

Tests that, given an input of 0 bytes, the `toNibbles` function returns a zero
length array.


```solidity
function test_toNibbles_zeroLengthInput_works() public;
```

### testDiff_toNibbles_succeeds

Test that the `toNibbles` function in the `Bytes` library is equivalent to the Yul
implementation.


```solidity
function testDiff_toNibbles_succeeds(bytes memory _input) public;
```

