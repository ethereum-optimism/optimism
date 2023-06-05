# FFIInterface
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/test/CommonTest.t.sol)

**Inherits:**
Test


## Functions
### getProveWithdrawalTransactionInputs


```solidity
function getProveWithdrawalTransactionInputs(Types.WithdrawalTransaction memory _tx)
    external
    returns (bytes32, bytes32, bytes32, bytes32, bytes[] memory);
```

### hashCrossDomainMessage


```solidity
function hashCrossDomainMessage(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external returns (bytes32);
```

### hashWithdrawal


```solidity
function hashWithdrawal(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external returns (bytes32);
```

### hashOutputRootProof


```solidity
function hashOutputRootProof(
    bytes32 _version,
    bytes32 _stateRoot,
    bytes32 _messagePasserStorageRoot,
    bytes32 _latestBlockhash
) external returns (bytes32);
```

### hashDepositTransaction


```solidity
function hashDepositTransaction(
    address _from,
    address _to,
    uint256 _mint,
    uint256 _value,
    uint64 _gas,
    bytes memory _data,
    uint64 _logIndex
) external returns (bytes32);
```

### encodeDepositTransaction


```solidity
function encodeDepositTransaction(Types.UserDepositTransaction calldata txn) external returns (bytes memory);
```

### encodeCrossDomainMessage


```solidity
function encodeCrossDomainMessage(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) external returns (bytes memory);
```

### decodeVersionedNonce


```solidity
function decodeVersionedNonce(uint256 nonce) external returns (uint256, uint256);
```

### getMerkleTrieFuzzCase


```solidity
function getMerkleTrieFuzzCase(string memory variant)
    external
    returns (bytes32, bytes memory, bytes memory, bytes[] memory);
```

