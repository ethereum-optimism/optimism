# IBondManager
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/dispute/IBondManager.sol)

The Bond Manager holds ether posted as a bond for a bond id.


## Functions
### post

Post a bond with a given id and owner.

*This function will revert if the provided bondId is already in use.*


```solidity
function post(bytes32 _bondId, address _bondOwner, uint256 _minClaimHold) external payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|
|`_bondOwner`|`address`|is the address that owns the bond.|
|`_minClaimHold`|`uint256`|is the minimum amount of time the owner must wait before reclaiming their bond.|


### seize

Seizes the bond with the given id.

*This function will revert if there is no bond at the given id.*


```solidity
function seize(bytes32 _bondId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|


### seizeAndSplit

Seizes the bond with the given id and distributes it to recipients.

*This function will revert if there is no bond at the given id.*


```solidity
function seizeAndSplit(bytes32 _bondId, address[] calldata _claimRecipients) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|
|`_claimRecipients`|`address[]`|is a set of addresses to split the bond amongst.|


### reclaim

Reclaims the bond of the bond owner.

*This function will revert if there is no bond at the given id.*


```solidity
function reclaim(bytes32 _bondId) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_bondId`|`bytes32`|is the id of the bond.|


