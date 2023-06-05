# Hashing_hashDepositTransaction_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Hashing.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testDiff_hashDepositTransaction_succeeds

Tests that hashDepositTransaction returns the correct hash in a simple case.


```solidity
function testDiff_hashDepositTransaction_succeeds(
    address _from,
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gas,
    bytes memory _data,
    uint64 _logIndex
) external;
```

