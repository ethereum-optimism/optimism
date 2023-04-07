# Clone
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/util/Clone.sol)

**Author:**
zefram.eth

Provides helper functions for reading immutable args from calldata


## Functions
### _getArgAddress

Reads an immutable arg with type address


```solidity
function _getArgAddress(uint256 argOffset) internal pure returns (address arg);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`argOffset`|`uint256`|The offset of the arg in the packed data|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`arg`|`address`|The arg value|


### _getArgUint256

Reads an immutable arg with type uint256


```solidity
function _getArgUint256(uint256 argOffset) internal pure returns (uint256 arg);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`argOffset`|`uint256`|The offset of the arg in the packed data|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`arg`|`uint256`|The arg value|


### _getArgUint256Array

Reads a uint256 array stored in the immutable args.


```solidity
function _getArgUint256Array(uint256 argOffset, uint64 arrLen) internal pure returns (uint256[] memory arr);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`argOffset`|`uint256`|The offset of the arg in the packed data|
|`arrLen`|`uint64`|Number of elements in the array|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`arr`|`uint256[]`|The array|


### _getArgUint64

Reads an immutable arg with type uint64


```solidity
function _getArgUint64(uint256 argOffset) internal pure returns (uint64 arg);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`argOffset`|`uint256`|The offset of the arg in the packed data|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`arg`|`uint64`|The arg value|


### _getArgUint8

Reads an immutable arg with type uint8


```solidity
function _getArgUint8(uint256 argOffset) internal pure returns (uint8 arg);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`argOffset`|`uint256`|The offset of the arg in the packed data|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`arg`|`uint8`|The arg value|


### _getImmutableArgsOffset


```solidity
function _getImmutableArgsOffset() internal pure returns (uint256 offset);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`offset`|`uint256`|The offset of the packed immutable args in calldata|


