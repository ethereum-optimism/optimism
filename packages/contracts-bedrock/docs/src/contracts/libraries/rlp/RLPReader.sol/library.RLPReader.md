# RLPReader
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/rlp/RLPReader.sol)

RLPReader is a library for parsing RLP-encoded byte arrays into Solidity types. Adapted
from Solidity-RLP (https://github.com/hamdiallam/Solidity-RLP) by Hamdi Allam with
various tweaks to improve readability.


## State Variables
### MAX_LIST_LENGTH
Max list length that this library will accept.


```solidity
uint256 internal constant MAX_LIST_LENGTH = 32;
```


## Functions
### toRLPItem

Converts bytes to a reference to memory position and length.


```solidity
function toRLPItem(bytes memory _in) internal pure returns (RLPItem memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bytes`|Input bytes to convert.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`RLPItem`|Output memory reference.|


### readList

Reads an RLP list value into a list of RLP items.


```solidity
function readList(RLPItem memory _in) internal pure returns (RLPItem[] memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`RLPItem`|RLP list value.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`RLPItem[]`|Decoded RLP list items.|


### readList

Reads an RLP list value into a list of RLP items.


```solidity
function readList(bytes memory _in) internal pure returns (RLPItem[] memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bytes`|RLP list value.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`RLPItem[]`|Decoded RLP list items.|


### readBytes

Reads an RLP bytes value into bytes.


```solidity
function readBytes(RLPItem memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`RLPItem`|RLP bytes value.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Decoded bytes.|


### readBytes

Reads an RLP bytes value into bytes.


```solidity
function readBytes(bytes memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`bytes`|RLP bytes value.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Decoded bytes.|


### readRawBytes

Reads the raw bytes of an RLP item.


```solidity
function readRawBytes(RLPItem memory _in) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`RLPItem`|RLP item to read.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Raw RLP bytes.|


### _decodeLength

Decodes the length of an RLP item.


```solidity
function _decodeLength(RLPItem memory _in) private pure returns (uint256, uint256, RLPItemType);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_in`|`RLPItem`|RLP item to decode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Offset of the encoded data.|
|`<none>`|`uint256`|Length of the encoded data.|
|`<none>`|`RLPItemType`|RLP item type (LIST_ITEM or DATA_ITEM).|


### _copy

Copies the bytes from a memory location.


```solidity
function _copy(MemoryPointer _src, uint256 _offset, uint256 _length) private pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_src`|`MemoryPointer`|   Pointer to the location to read from.|
|`_offset`|`uint256`|Offset to start reading from.|
|`_length`|`uint256`|Number of bytes to read.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|Copied bytes.|


## Structs
### RLPItem
Struct representing an RLP item.


```solidity
struct RLPItem {
    uint256 length;
    MemoryPointer ptr;
}
```

## Enums
### RLPItemType
RLP item types.


```solidity
enum RLPItemType {
    DATA_ITEM,
    LIST_ITEM
}
```

