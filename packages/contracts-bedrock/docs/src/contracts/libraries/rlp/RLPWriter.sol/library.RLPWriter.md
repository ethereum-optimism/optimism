# RLPWriter
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/rlp/RLPWriter.sol)

**Author:**
RLPWriter is a library for encoding Solidity types to RLP bytes. Adapted from Bakaoh's
RLPEncode library (https://github.com/bakaoh/solidity-rlp-encode) with minor
modifications to improve legibility.


## Functions
### writeBytes

RLP encodes a byte string.


```solidity
function writeBytes(bytes memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bytes`|The byte string to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded string in bytes.|


### writeList

RLP encodes a list of RLP encoded byte byte strings.


```solidity
function writeList(bytes[] memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bytes[]`|The list of RLP encoded byte strings.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded list of items in bytes.|


### writeString

RLP encodes a string.


```solidity
function writeString(string memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`string`|The string to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded string in bytes.|


### writeAddress

RLP encodes an address.


```solidity
function writeAddress(address _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`address`|The address to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded address in bytes.|


### writeUint

RLP encodes a uint.


```solidity
function writeUint(uint256 _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`uint256`|The uint256 to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded uint256 in bytes.|


### writeBool

RLP encodes a bool.


```solidity
function writeBool(bool _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bool`|The bool to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The RLP encoded bool in bytes.|


### _writeLength

Encode the first byte and then the `len` in binary form if `length` is more than 55.


```solidity
function _writeLength(uint256 _len, uint256 _offset) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_len`|`uint256`|   The length of the string or the payload.|
|`_offset`|`uint256`|128 if item is string, 192 if item is list.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|RLP encoded bytes.|


### _toBinary

Encode integer in big endian binary form with no leading zeroes.


```solidity
function _toBinary(uint256 _x) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_x`|`uint256`|The integer to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|RLP encoded bytes.|


### _memcpy

Copies a piece of memory to another location.


```solidity
function _memcpy(uint256 _dest, uint256 _src, uint256 _len) private pure;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_dest`|`uint256`|Destination location.|
|`_src`|`uint256`| Source location.|
|`_len`|`uint256`| Length of memory to copy.|


### _flatten

Flattens a list of byte strings into one byte string.


```solidity
function _flatten(bytes[] memory _list) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_list`|`bytes[]`|List of byte strings to flatten.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|The flattened byte string.|


