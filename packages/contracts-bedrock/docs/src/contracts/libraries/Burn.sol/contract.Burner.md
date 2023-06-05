# Burner
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Burn.sol)

Burner self-destructs on creation and sends all ETH to itself, removing all ETH given to
the contract from the circulating supply. Self-destructing is the only way to remove ETH
from the circulating supply.


## Functions
### constructor


```solidity
constructor() payable;
```

