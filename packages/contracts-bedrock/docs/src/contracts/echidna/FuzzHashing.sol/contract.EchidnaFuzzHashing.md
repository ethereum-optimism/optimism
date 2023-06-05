# EchidnaFuzzHashing
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/echidna/FuzzHashing.sol)


## State Variables
### failedCrossDomainHashHighVersion

```solidity
bool internal failedCrossDomainHashHighVersion;
```


### failedCrossDomainHashV0

```solidity
bool internal failedCrossDomainHashV0;
```


### failedCrossDomainHashV1

```solidity
bool internal failedCrossDomainHashV1;
```


## Functions
### testHashCrossDomainMessageHighVersion

Takes the necessary parameters to perform a cross domain hash with a randomly
generated version. Only schema versions 0 and 1 are supported and all others should revert.


```solidity
function testHashCrossDomainMessageHighVersion(
    uint16 _version,
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) public;
```

### testHashCrossDomainMessageV0

Takes the necessary parameters to perform a cross domain hash using the v0 schema
and compares the output of a call to the unversioned function to the v0 function directly


```solidity
function testHashCrossDomainMessageV0(
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) public;
```

### testHashCrossDomainMessageV1

Takes the necessary parameters to perform a cross domain hash using the v1 schema
and compares the output of a call to the unversioned function to the v1 function directly


```solidity
function testHashCrossDomainMessageV1(
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) public;
```

### echidna_hash_xdomain_msg_high_version


```solidity
function echidna_hash_xdomain_msg_high_version() public view returns (bool);
```

### echidna_hash_xdomain_msg_0


```solidity
function echidna_hash_xdomain_msg_0() public view returns (bool);
```

### echidna_hash_xdomain_msg_1


```solidity
function echidna_hash_xdomain_msg_1() public view returns (bool);
```

