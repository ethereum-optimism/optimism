# IVersioned
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/dispute/IVersioned.sol)

An interface for semantically versioned contracts.


## Functions
### version

Returns the semantic version of the contract


```solidity
function version() external pure returns (string memory _version);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_version`|`string`|The semantic version of the contract|


