# Encoding_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Encoding.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testFuzz_nonceVersioning_succeeds


```solidity
function testFuzz_nonceVersioning_succeeds(uint240 _nonce, uint16 _version) external;
```

### testDiff_decodeVersionedNonce_succeeds


```solidity
function testDiff_decodeVersionedNonce_succeeds(uint240 _nonce, uint16 _version) external;
```

### testDiff_encodeCrossDomainMessage_succeeds


```solidity
function testDiff_encodeCrossDomainMessage_succeeds(
    uint240 _nonce,
    uint8 _version,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

### testFuzz_encodeCrossDomainMessageV0_matchesLegacy_succeeds


```solidity
function testFuzz_encodeCrossDomainMessageV0_matchesLegacy_succeeds(
    uint240 _nonce,
    address _sender,
    address _target,
    bytes memory _data
) external;
```

### testDiff_encodeDepositTransaction_succeeds


```solidity
function testDiff_encodeDepositTransaction_succeeds(
    address _from,
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gas,
    bool isCreate,
    bytes memory _data,
    uint64 _logIndex
) external;
```

