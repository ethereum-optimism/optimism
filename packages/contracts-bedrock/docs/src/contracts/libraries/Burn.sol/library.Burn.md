# Burn
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Burn.sol)

Utilities for burning stuff.


## Functions
### eth

Burns a given amount of ETH.


```solidity
function eth(uint256 _amount) internal;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_amount`|`uint256`|Amount of ETH to burn.|


### gas

Burns a given amount of gas.


```solidity
function gas(uint256 _amount) internal view;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_amount`|`uint256`|Amount of gas to burn.|


