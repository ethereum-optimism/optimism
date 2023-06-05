# CrossDomainOwnable2
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/CrossDomainOwnable2.sol)

**Inherits:**
Ownable

This contract extends the OpenZeppelin `Ownable` contract for L2 contracts to be owned
by contracts on L1. Note that this contract is meant to be used with systems that use
the CrossDomainMessenger system. It will not work if the OptimismPortal is used
directly.


## Functions
### _checkOwner

Overrides the implementation of the `onlyOwner` modifier to check that the unaliased
`xDomainMessageSender` is the owner of the contract. This value is set to the caller
of the L1CrossDomainMessenger.


```solidity
function _checkOwner() internal view override;
```

