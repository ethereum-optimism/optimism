# FeeVault_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/FeeVault.t.sol)

**Inherits:**
[Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.Bridge_Initializer.md)


## State Variables
### baseFeeVault

```solidity
BaseFeeVault baseFeeVault = BaseFeeVault(payable(Predeploys.BASE_FEE_VAULT));
```


### l1FeeVault

```solidity
L1FeeVault l1FeeVault = L1FeeVault(payable(Predeploys.L1_FEE_VAULT));
```


### recipient

```solidity
address constant recipient = address(0x10000);
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_minWithdrawalAmount_succeeds


```solidity
function test_minWithdrawalAmount_succeeds() external;
```

