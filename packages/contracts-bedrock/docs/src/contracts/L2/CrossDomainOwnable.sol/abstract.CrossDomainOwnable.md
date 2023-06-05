# CrossDomainOwnable
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/L2/CrossDomainOwnable.sol)

**Inherits:**
Ownable

This contract extends the OpenZeppelin `Ownable` contract for L2 contracts to be owned
by contracts on L1. Note that this contract is only safe to be used if the
CrossDomainMessenger system is bypassed and the caller on L1 is calling the
OptimismPortal directly.


## Functions
### _checkOwner

Overrides the implementation of the `onlyOwner` modifier to check that the unaliased
`msg.sender` is the owner of the contract.


```solidity
function _checkOwner() internal view override;
```

