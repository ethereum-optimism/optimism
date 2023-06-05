# L1BlockTest
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/L1Block.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## State Variables
### lb

```solidity
L1Block lb;
```


### depositor

```solidity
address depositor;
```


### NON_ZERO_HASH

```solidity
bytes32 immutable NON_ZERO_HASH = keccak256(abi.encode(1));
```


## Functions
### setUp


```solidity
function setUp() public virtual override;
```

### testFuzz_updatesValues_succeeds


```solidity
function testFuzz_updatesValues_succeeds(
    uint64 n,
    uint64 t,
    uint256 b,
    bytes32 h,
    uint64 s,
    bytes32 bt,
    uint256 fo,
    uint256 fs
) external;
```

### test_number_succeeds


```solidity
function test_number_succeeds() external;
```

### test_timestamp_succeeds


```solidity
function test_timestamp_succeeds() external;
```

### test_basefee_succeeds


```solidity
function test_basefee_succeeds() external;
```

### test_hash_succeeds


```solidity
function test_hash_succeeds() external;
```

### test_sequenceNumber_succeeds


```solidity
function test_sequenceNumber_succeeds() external;
```

### test_updateValues_succeeds


```solidity
function test_updateValues_succeeds() external;
```

