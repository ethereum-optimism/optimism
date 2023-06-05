# ClaimHash
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/DisputeTypes.sol)

A claim hash represents a hash of a claim and a position within the game tree.

*Keccak hash of abi.encodePacked(Claim, Position);*


```solidity
type ClaimHash is bytes32;
```

