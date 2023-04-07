# IAttestationDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/interfaces/IAttestationDisputeGame.sol)

**Inherits:**
[IDisputeGame](/src/interfaces/IDisputeGame.sol/interface.IDisputeGame.md)

The interface for an attestation-based DisputeGame meant to contest output
proposals in Optimism's `L2OutputOracle` contract.


## Functions
### challenges

A mapping of addresses from the `signerSet` to booleans signifying whether
or not they have authorized the `rootClaim` to be invalidated.


```solidity
function challenges(address challenger) external view returns (bool _challenged);
```

### signerSet

The signer set consists of authorized public keys that may challenge the `rootClaim`.


```solidity
function signerSet() external view returns (address[] memory _signers);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`_signers`|`address[]`|An array of authorized signers.|


### signatureThreshold

The amount of signatures required to successfully challenge the `rootClaim`
output proposal. Once this threshold is met by members of the `signerSet`
calling `challenge`, the game will be resolved to `CHALLENGER_WINS`.


```solidity
function signatureThreshold() external view returns (uint16 _signatureThreshold);
```

### l2BlockNumber

Returns the L2 Block Number that the `rootClaim` commits to. Exists within the `extraData`.


```solidity
function l2BlockNumber() external view returns (uint256 _l2BlockNumber);
```

### challenge

Challenge the `rootClaim`.

*- If the `ecrecover`ed address that created the signature is not a part of the
signer set returned by `signerSet`, this function should revert.
- If the `ecrecover`ed address that created the signature is not the msg.sender,
this function should revert.
- If the signature provided is the signature that breaches the signature threshold,
the function should call the `resolve` function to resolve the game as `CHALLENGER_WINS`.
- When the game resolves, the bond attached to the root claim should be distributed among
the signers who participated in challenging the invalid claim.*


```solidity
function challenge(bytes calldata signature) external;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`signature`|`bytes`|An EIP-712 signature committing to the `rootClaim` and `l2BlockNumber` (within the `extraData`) from a key that exists within the `signerSet`.|


