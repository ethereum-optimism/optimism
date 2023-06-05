# CrossDomainMessengerLegacySpacer0
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/CrossDomainMessenger.sol)

Contract only exists to add a spacer to the CrossDomainMessenger where the
libAddressManager variable used to exist. Must be the first contract in the inheritance
tree of the CrossDomainMessenger.


## State Variables
### spacer_0_0_20
Spacer for backwards compatibility.


```solidity
address private spacer_0_0_20;
```


