# NotOwner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/c6ae546047e96fbfd2d0f78febba2885aab34f5f/src/types/Errors.sol)

Thrown when a function that is protected by the `onlyOwner` modifier is called from an account
other than the owner.


```solidity
error NotOwner();
```

