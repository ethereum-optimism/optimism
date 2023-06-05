# Hashing_hashOutputRootProof_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Hashing.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testDiff_hashOutputRootProof_succeeds

Tests that hashOutputRootProof returns the correct hash in a simple case.


```solidity
function testDiff_hashOutputRootProof_succeeds(
    bytes32 _version,
    bytes32 _stateRoot,
    bytes32 _messagePasserStorageRoot,
    bytes32 _latestBlockhash
) external;
```

