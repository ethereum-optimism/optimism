# IValididtyDisputeGame
[Git Source](https://github.com/ethereum-optimism/optimism/blob/eaf1cde5896035c9ff0d32731da1e103f2f1c693/src/interfaces/IValidityDisputeGame.sol)

**Inherits:**
[IDisputeGame](/src/interfaces/IDisputeGame.sol/interface.IDisputeGame.md)

The interface for a validity proof backed dispute game.


## Functions
### prove

Proves the root claim

*Underneath the hood, the separate implementations will unpack the `data` differently
due to the different proof verification algorithms for SNARKs, PLONKs, etc.*


```solidity
function prove(bytes calldata input) external;
```

