# Hashing_hashCrossDomainMessage_Test
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/Hashing.t.sol)

**Inherits:**
[CommonTest](/contracts/test/CommonTest.t.sol/contract.CommonTest.md)


## Functions
### testDiff_hashCrossDomainMessage_succeeds

Tests that hashCrossDomainMessage returns the correct hash in a simple case.


```solidity
function testDiff_hashCrossDomainMessage_succeeds(
    uint240 _nonce,
    uint16 _version,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

### testFuzz_hashCrossDomainMessageV0_matchesLegacy_succeeds

Tests that hashCrossDomainMessageV0 matches the hash of the legacy encoding.


```solidity
function testFuzz_hashCrossDomainMessageV0_matchesLegacy_succeeds(
    address _target,
    address _sender,
    bytes memory _message,
    uint256 _messageNonce
) external;
```

