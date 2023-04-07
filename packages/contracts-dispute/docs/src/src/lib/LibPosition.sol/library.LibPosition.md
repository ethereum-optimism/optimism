# LibPosition
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/lib/LibPosition.sol)

**Author:**
clabby <https://github.com/clabby>

This library contains helper functions for working with the `Position` type.


## Functions
### wrap


```solidity
function wrap(uint128 _depth, uint128 _indexAtDepth) internal pure returns (Position _position);
```

### depth

Pulls the `depth` out of a packed `Position` type.


```solidity
function depth(Position position) internal pure returns (uint128 _depth);
```

### indexAtDepth

Pulls the `indexAtDepth` out of a packed `Position` type.


```solidity
function indexAtDepth(Position position) internal pure returns (uint128 _indexAtDepth);
```

### left

Get the position to the left of `position`.


```solidity
function left(Position position) internal pure returns (Position _left);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`position`|`Position`|The position to get the left position of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_left`|`Position`|The position to the left of `position`.|


### right

Get the position to the right of `position`.


```solidity
function right(Position position) internal pure returns (Position _right);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`position`|`Position`|The position to get the right position of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_right`|`Position`|The position to the right of `position`.|


### parent

Get the parent position of `position`.


```solidity
function parent(Position position) internal pure returns (Position _parent);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`position`|`Position`|The position to get the parent position of.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_parent`|`Position`|The parent position of `position`.|


### rightIndex

Get the deepest, right most index relative to the `position`.


```solidity
function rightIndex(Position position, uint256 maxDepth) internal pure returns (uint128 _rightIndex);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`position`|`Position`|The position to get the relative deepest, right most index of.|
|`maxDepth`|`uint256`|The maximum depth of the game.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_rightIndex`|`uint128`|The deepest, right most index relative to the `position`. TODO: Optimize; No need to update the full position in the sub loop.|


### attack

Get the attack position relative to `position`.


```solidity
function attack(Position position) internal pure returns (Position _attack);
```

### defend

Get the defend position of `position`.


```solidity
function defend(Position position) internal pure returns (Position _defend);
```

