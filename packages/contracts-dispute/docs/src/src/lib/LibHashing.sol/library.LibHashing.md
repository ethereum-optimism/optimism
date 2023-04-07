# LibHashing
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/lib/LibHashing.sol)

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


