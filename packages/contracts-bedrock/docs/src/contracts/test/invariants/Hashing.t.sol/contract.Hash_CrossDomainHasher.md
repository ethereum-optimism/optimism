# Hash_CrossDomainHasher
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/invariants/Hashing.t.sol)


## State Variables
### failedCrossDomainHashHighVersion

```solidity
bool public failedCrossDomainHashHighVersion;
```


### failedCrossDomainHashV0

```solidity
bool public failedCrossDomainHashV0;
```


### failedCrossDomainHashV1

```solidity
bool public failedCrossDomainHashV1;
```


## Functions
### hashCrossDomainMessageHighVersion

Takes the necessary parameters to perform a cross domain hash with a randomly
generated version. Only schema versions 0 and 1 are supported and all others should revert.


```solidity
function hashCrossDomainMessageHighVersion(
    uint16 _version,
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

### hashCrossDomainMessageV0

Takes the necessary parameters to perform a cross domain hash using the v0 schema
and compares the output of a call to the unversioned function to the v0 function directly


```solidity
function hashCrossDomainMessageV0(
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

### hashCrossDomainMessageV1

Takes the necessary parameters to perform a cross domain hash using the v1 schema
and compares the output of a call to the unversioned function to the v1 function directly


```solidity
function hashCrossDomainMessageV1(
    uint240 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external;
```

