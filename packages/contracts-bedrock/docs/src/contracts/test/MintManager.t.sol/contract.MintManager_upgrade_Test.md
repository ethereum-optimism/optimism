# MintManager_upgrade_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/MintManager.t.sol)

**Inherits:**
[MintManager_Initializer](/contracts/test/MintManager.t.sol/contract.MintManager_Initializer.md)


## Functions
### test_upgrade_fromOwner_succeeds

Tests that the owner can upgrade the mint manager.


```solidity
function test_upgrade_fromOwner_succeeds() external;
```

### test_upgrade_fromNotOwner_reverts

Tests that the upgrade function reverts when called by a non-owner.


```solidity
function test_upgrade_fromNotOwner_reverts() external;
```

### test_upgrade_toZeroAddress_reverts

Tests that the upgrade function reverts when attempting to update to the zero
address, even if the caller is the owner.


```solidity
function test_upgrade_toZeroAddress_reverts() external;
```

