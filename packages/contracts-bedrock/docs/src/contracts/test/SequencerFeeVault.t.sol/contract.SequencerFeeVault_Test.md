# SequencerFeeVault_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/SequencerFeeVault.t.sol)

**Inherits:**
[Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.Bridge_Initializer.md)


## State Variables
### vault

```solidity
SequencerFeeVault vault = SequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
```


### recipient

```solidity
address constant recipient = address(256);
```


## Functions
### setUp


```solidity
function setUp() public override;
```

### test_minWithdrawalAmount_succeeds


```solidity
function test_minWithdrawalAmount_succeeds() external;
```

### test_constructor_succeeds


```solidity
function test_constructor_succeeds() external;
```

### test_receive_succeeds


```solidity
function test_receive_succeeds() external;
```

### test_withdraw_notEnough_reverts


```solidity
function test_withdraw_notEnough_reverts() external;
```

### test_withdraw_succeeds


```solidity
function test_withdraw_succeeds() external;
```

## Events
### Withdrawal

```solidity
event Withdrawal(uint256 value, address to, address from);
```

