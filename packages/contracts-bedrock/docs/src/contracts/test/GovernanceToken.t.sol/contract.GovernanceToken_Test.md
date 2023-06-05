# GovernanceToken_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/GovernanceToken.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## State Variables
### owner

```solidity
address constant owner = address(0x1234);
```


### rando

```solidity
address constant rando = address(0x5678);
```


### gov

```solidity
GovernanceToken internal gov;
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_mint_fromOwner_succeeds


```solidity
function test_mint_fromOwner_succeeds() external;
```

### test_mint_fromNotOwner_reverts


```solidity
function test_mint_fromNotOwner_reverts() external;
```

### test_burn_succeeds


```solidity
function test_burn_succeeds() external;
```

### test_burnFrom_succeeds


```solidity
function test_burnFrom_succeeds() external;
```

### test_transfer_succeeds


```solidity
function test_transfer_succeeds() external;
```

### test_approve_succeeds


```solidity
function test_approve_succeeds() external;
```

### test_transferFrom_succeeds


```solidity
function test_transferFrom_succeeds() external;
```

### test_increaseAllowance_succeeds


```solidity
function test_increaseAllowance_succeeds() external;
```

### test_decreaseAllowance_succeeds


```solidity
function test_decreaseAllowance_succeeds() external;
```

