# IOwnable
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/interfaces/IOwnable.sol)

An interface for ownable contracts.


## Functions
### owner

Returns the owner of the contract


```solidity
function owner() external view returns (address);
```

### transferOwnership

Transfer ownership to the passed address

*May only be called by the `owner`.*


```solidity
function transferOwnership(address newOwner) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`newOwner`|`address`|The address to transfer ownership to|


