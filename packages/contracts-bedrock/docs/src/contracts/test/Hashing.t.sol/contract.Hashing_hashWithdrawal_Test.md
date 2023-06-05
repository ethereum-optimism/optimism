# Hashing_hashWithdrawal_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Hashing.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testDiff_hashWithdrawal_succeeds

Tests that hashWithdrawal returns the correct hash in a simple case.


```solidity
function testDiff_hashWithdrawal_succeeds(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

