# Hashing
[Git Source](https://github.com/ethereum-optimism/cannon-v2-contracts/blob/896a9e7a2e7769b1273deb0b0a9ed4c533f56f75/src/lib/LibHashing.sol)

This library contains all of the hashing utilities used in the Cannon contracts.


## Functions
### hashGindexClaim

Hashes a generalized index and a claim together.


```solidity
function hashGindexClaim(Types.Gindex gindex, Types.Claim claim)
    internal
    pure
    returns (Types.GindexClaim gindexClaim);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`gindex`|`Gindex.Types`|A generalized index.|
|`claim`|`Claim.Types`|A [Types.Claim].|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`gindexClaim`|`GindexClaim.Types`|A [Types.GIndexClaim] representing a generalized index and a claim hashed together.|


