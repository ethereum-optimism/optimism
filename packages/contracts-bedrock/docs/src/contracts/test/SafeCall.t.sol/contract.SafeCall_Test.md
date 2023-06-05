# SafeCall_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/SafeCall.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testFuzz_send_succeeds


```solidity
function testFuzz_send_succeeds(address from, address to, uint256 gas, uint64 value) external;
```

### testFuzz_call_succeeds


```solidity
function testFuzz_call_succeeds(address from, address to, uint256 gas, uint64 value, bytes memory data) external;
```

### testFuzz_callWithMinGas_hasEnough_succeeds


```solidity
function testFuzz_callWithMinGas_hasEnough_succeeds(
    address from,
    address to,
    uint64 minGas,
    uint64 value,
    bytes memory data
) external;
```

### test_callWithMinGas_noLeakageLow_succeeds


```solidity
function test_callWithMinGas_noLeakageLow_succeeds() external;
```

### test_callWithMinGas_noLeakageHigh_succeeds


```solidity
function test_callWithMinGas_noLeakageHigh_succeeds() external;
```

