# MerkleTrie_get_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/MerkleTrie.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### test_get_validProof1_succeeds


```solidity
function test_get_validProof1_succeeds() external;
```

### test_get_validProof2_succeeds


```solidity
function test_get_validProof2_succeeds() external;
```

### test_get_validProof3_succeeds


```solidity
function test_get_validProof3_succeeds() external;
```

### test_get_validProof4_succeeds


```solidity
function test_get_validProof4_succeeds() external;
```

### test_get_validProof5_succeeds


```solidity
function test_get_validProof5_succeeds() external;
```

### test_get_validProof6_succeeds


```solidity
function test_get_validProof6_succeeds() external;
```

### test_get_validProof7_succeeds


```solidity
function test_get_validProof7_succeeds() external;
```

### test_get_validProof8_succeeds


```solidity
function test_get_validProof8_succeeds() external;
```

### test_get_validProof9_succeeds


```solidity
function test_get_validProof9_succeeds() external;
```

### test_get_validProof10_succeeds


```solidity
function test_get_validProof10_succeeds() external;
```

### test_get_nonexistentKey1_reverts


```solidity
function test_get_nonexistentKey1_reverts() external;
```

### test_get_nonexistentKey2_reverts


```solidity
function test_get_nonexistentKey2_reverts() external;
```

### test_get_wrongKeyProof_reverts


```solidity
function test_get_wrongKeyProof_reverts() external;
```

### test_get_corruptedProof_reverts


```solidity
function test_get_corruptedProof_reverts() external;
```

### test_get_invalidDataRemainder_reverts


```solidity
function test_get_invalidDataRemainder_reverts() external;
```

### test_get_invalidInternalNodeHash_reverts


```solidity
function test_get_invalidInternalNodeHash_reverts() external;
```

### test_get_zeroBranchValueLength_reverts


```solidity
function test_get_zeroBranchValueLength_reverts() external;
```

### test_get_zeroLengthKey_reverts


```solidity
function test_get_zeroLengthKey_reverts() external;
```

### test_get_smallerPathThanKey1_reverts


```solidity
function test_get_smallerPathThanKey1_reverts() external;
```

### test_get_smallerPathThanKey2_reverts


```solidity
function test_get_smallerPathThanKey2_reverts() external;
```

### test_get_extraProofElements_reverts


```solidity
function test_get_extraProofElements_reverts() external;
```

### testFuzz_get_validProofs_succeeds

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_validProofs_succeeds(bytes4) external;
```

### testFuzz_get_invalidRoot_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_invalidRoot_reverts(bytes4) external;
```

### testFuzz_get_extraProofElements_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_extraProofElements_reverts(bytes4) external;
```

### testFuzz_get_invalidLargeInternalHash_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_invalidLargeInternalHash_reverts(bytes4) external;
```

### testFuzz_get_invalidInternalNodeHash_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_invalidInternalNodeHash_reverts(bytes4) external;
```

### testFuzz_get_corruptedProof_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_corruptedProof_reverts(bytes4) external;
```

### testFuzz_get_invalidDataRemainder_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_invalidDataRemainder_reverts(bytes4) external;
```

### testFuzz_get_prefixedValidKey_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_prefixedValidKey_reverts(bytes4) external;
```

### testFuzz_get_emptyKey_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_emptyKey_reverts(bytes4) external;
```

### testFuzz_get_partialProof_reverts

The `bytes4` parameter is to enable parallel fuzz runs; it is ignored.


```solidity
function testFuzz_get_partialProof_reverts(bytes4) external;
```

