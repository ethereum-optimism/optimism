# LibHashing
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/lib/LibHashing.sol)

**Author:**
clabby <https://github.com/clabby>

This library contains all of the hashing utilities used in the Cannon contracts.


## Functions
### hashClaimPos

Hashes a claim and a position together.


```solidity
function hashClaimPos(Claim claim, Position position) internal pure returns (ClaimHash claimHash);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`claim`|`Claim`|A Claim type.|
|`position`|`Position`|The position of `claim`.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`claimHash`|`ClaimHash`|A hash of abi.encodePacked(claim, position);|


