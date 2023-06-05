# AddressAliasHelper
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/vendor/AddressAliasHelper.sol)


## State Variables
### offset

```solidity
uint160 constant offset = uint160(0x1111000000000000000000000000000000001111);
```


## Functions
### applyL1ToL2Alias

Utility function that converts the address in the L1 that submitted a tx to
the inbox to the msg.sender viewed in the L2


```solidity
function applyL1ToL2Alias(address l1Address) internal pure returns (address l2Address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`l1Address`|`address`|the address in the L1 that triggered the tx to L2|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`l2Address`|`address`|L2 address as viewed in msg.sender|


### undoL1ToL2Alias

Utility function that converts the msg.sender viewed in the L2 to the
address in the L1 that submitted a tx to the inbox


```solidity
function undoL1ToL2Alias(address l2Address) internal pure returns (address l1Address);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`l2Address`|`address`|L2 address as viewed in msg.sender|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`l1Address`|`address`|the address in the L1 that triggered the tx to L2|


