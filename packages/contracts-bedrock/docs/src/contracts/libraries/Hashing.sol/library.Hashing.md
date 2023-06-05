# Hashing
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Hashing.sol)

Hashing handles Optimism's various different hashing schemes.


## Functions
### hashDepositTransaction

Computes the hash of the RLP encoded L2 transaction that would be generated when a
given deposit is sent to the L2 system. Useful for searching for a deposit in the L2
system.


```solidity
function hashDepositTransaction(Types.UserDepositTransaction memory _tx) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_tx`|`UserDepositTransaction.Types`|User deposit transaction to hash.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hash of the RLP encoded L2 deposit transaction.|


### hashDepositSource

Computes the deposit transaction's "source hash", a value that guarantees the hash
of the L2 transaction that corresponds to a deposit is unique and is
deterministically generated from L1 transaction data.


```solidity
function hashDepositSource(bytes32 _l1BlockHash, uint256 _logIndex) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_l1BlockHash`|`bytes32`|Hash of the L1 block where the deposit was included.|
|`_logIndex`|`uint256`|   The index of the log that created the deposit transaction.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hash of the deposit transaction's "source hash".|


### hashCrossDomainMessage

Hashes the cross domain message based on the version that is encoded into the
message nonce.


```solidity
function hashCrossDomainMessage(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_nonce`|`uint256`|   Message nonce with version encoded into the first two bytes.|
|`_sender`|`address`|  Address of the sender of the message.|
|`_target`|`address`|  Address of the target of the message.|
|`_value`|`uint256`|   ETH value to send to the target.|
|`_gasLimit`|`uint256`|Gas limit to use for the message.|
|`_data`|`bytes`|    Data to send with the message.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hashed cross domain message.|


### hashCrossDomainMessageV0

Hashes a cross domain message based on the V0 (legacy) encoding.


```solidity
function hashCrossDomainMessageV0(address _target, address _sender, bytes memory _data, uint256 _nonce)
    internal
    pure
    returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|Address of the target of the message.|
|`_sender`|`address`|Address of the sender of the message.|
|`_data`|`bytes`|  Data to send with the message.|
|`_nonce`|`uint256`| Message nonce.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hashed cross domain message.|


### hashCrossDomainMessageV1

Hashes a cross domain message based on the V1 (current) encoding.


```solidity
function hashCrossDomainMessageV1(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_nonce`|`uint256`|   Message nonce.|
|`_sender`|`address`|  Address of the sender of the message.|
|`_target`|`address`|  Address of the target of the message.|
|`_value`|`uint256`|   ETH value to send to the target.|
|`_gasLimit`|`uint256`|Gas limit to use for the message.|
|`_data`|`bytes`|    Data to send with the message.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hashed cross domain message.|


### hashWithdrawal

Derives the withdrawal hash according to the encoding in the L2 Withdrawer contract


```solidity
function hashWithdrawal(Types.WithdrawalTransaction memory _tx) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_tx`|`WithdrawalTransaction.Types`|Withdrawal transaction to hash.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hashed withdrawal transaction.|


### hashOutputRootProof

Hashes the various elements of an output root proof into an output root hash which
can be used to check if the proof is valid.


```solidity
function hashOutputRootProof(Types.OutputRootProof memory _outputRootProof) internal pure returns (bytes32);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_outputRootProof`|`OutputRootProof.Types`|Output root proof which should hash to an output root.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes32`|Hashed output root proof.|


