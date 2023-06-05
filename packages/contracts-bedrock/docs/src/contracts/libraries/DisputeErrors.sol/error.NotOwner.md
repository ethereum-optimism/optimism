# NotOwner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/DisputeErrors.sol)

Thrown when a function that is protected by the `onlyOwner` modifier
is called from an account other than the owner.


```solidity
error NotOwner();
```

