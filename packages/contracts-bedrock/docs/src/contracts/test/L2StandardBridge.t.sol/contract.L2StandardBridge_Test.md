# L2StandardBridge_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L2StandardBridge.t.sol)

**Inherits:**
[Bridge_Initializer](/contracts/test/CommonTest.t.sol/contract.Bridge_Initializer.md)


## Functions
### test_initialize_succeeds


```solidity
function test_initialize_succeeds() external;
```

### test_receive_succeeds


```solidity
function test_receive_succeeds() external;
```

### test_withdraw_insufficientValue_reverts


```solidity
function test_withdraw_insufficientValue_reverts() external;
```

### test_withdraw_ether_succeeds

Use the legacy `withdraw` interface on the L2StandardBridge to
withdraw ether from L2 to L1.


```solidity
function test_withdraw_ether_succeeds() external;
```

