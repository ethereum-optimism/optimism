# MintManager_mint_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/MintManager.t.sol)

**Inherits:**
[MintManager_Initializer](/contracts/test/MintManager.t.sol/contract.MintManager_Initializer.md)


## Functions
### test_mint_fromOwner_succeeds

Tests that the mint function properly mints tokens when called by the owner.


```solidity
function test_mint_fromOwner_succeeds() external;
```

### test_mint_fromNotOwner_reverts

Tests that the mint function reverts when called by a non-owner.


```solidity
function test_mint_fromNotOwner_reverts() external;
```

### test_mint_afterPeriodElapsed_succeeds

Tests that the mint function properly mints tokens when called by the owner a second
time after the mint period has elapsed.


```solidity
function test_mint_afterPeriodElapsed_succeeds() external;
```

### test_mint_beforePeriodElapsed_reverts

Tests that the mint function always reverts when called before the mint period has
elapsed, even if the caller is the owner.


```solidity
function test_mint_beforePeriodElapsed_reverts() external;
```

### test_mint_moreThanCap_reverts

Tests that the owner cannot mint more than the mint cap.


```solidity
function test_mint_moreThanCap_reverts() external;
```

