# Bytes
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Bytes.sol)

Bytes is a library for manipulating byte arrays.


## Functions
### slice

Slices a byte array with a given starting index and length. Returns a new byte array
as opposed to a pointer to the original array. Will throw if trying to slice more
bytes than exist in the array.


```solidity
function slice(bytes memory _bytes, uint256 _start, uint256 _length) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bytes`|`bytes`|Byte array to slice.|
|`_start`|`uint256`|Starting index of the slice.|
|`_length`|`uint256`|Length of the slice.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Slice of the input byte array.|


### slice

Slices a byte array with a given starting index up to the end of the original byte
array. Returns a new array rathern than a pointer to the original.


```solidity
function slice(bytes memory _bytes, uint256 _start) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bytes`|`bytes`|Byte array to slice.|
|`_start`|`uint256`|Starting index of the slice.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Slice of the input byte array.|


### toNibbles

Converts a byte array into a nibble array by splitting each byte into two nibbles.
Resulting nibble array will be exactly twice as long as the input byte array.


```solidity
function toNibbles(bytes memory _bytes) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bytes`|`bytes`|Input byte array to convert.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Resulting nibble array.|


### equal

Compares two byte arrays by comparing their keccak256 hashes.


```solidity
function equal(bytes memory _bytes, bytes memory _other) internal pure returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bytes`|`bytes`|First byte array to compare.|
|`_other`|`bytes`|Second byte array to compare.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|True if the two byte arrays are equal, false otherwise.|


