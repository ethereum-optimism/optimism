# Encoding
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/libraries/Encoding.sol)

Encoding handles Optimism's various different encoding schemes.


## Functions
### encodeDepositTransaction

RLP encodes the L2 transaction that would be generated when a given deposit is sent
to the L2 system. Useful for searching for a deposit in the L2 system. The
transaction is prefixed with 0x7e to identify its EIP-2718 type.


```solidity
function encodeDepositTransaction(Types.UserDepositTransaction memory _tx) internal pure returns (bytes memory);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_tx`|`UserDepositTransaction.Types`|User deposit transaction to encode.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bytes`|RLP encoded L2 deposit transaction.|


### encodeCrossDomainMessage

Encodes the cross domain message based on the version that is encoded into the
message nonce.


```solidity
function encodeCrossDomainMessage(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) internal pure returns (bytes memory);
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
|`<none>`|`bytes`|Encoded cross domain message.|


### encodeCrossDomainMessageV0

Encodes a cross domain message based on the V0 (legacy) encoding.


```solidity
function encodeCrossDomainMessageV0(address _target, address _sender, bytes memory _data, uint256 _nonce)
    internal
    pure
    returns (bytes memory);
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
|`<none>`|`bytes`|Encoded cross domain message.|


### encodeCrossDomainMessageV1

Encodes a cross domain message based on the V1 (current) encoding.


```solidity
function encodeCrossDomainMessageV1(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _gasLimit,
    bytes memory _data
) internal pure returns (bytes memory);
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
|`<none>`|`bytes`|Encoded cross domain message.|


### encodeVersionedNonce

Adds a version number into the first two bytes of a message nonce.


```solidity
function encodeVersionedNonce(uint240 _nonce, uint16 _version) internal pure returns (uint256);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_nonce`|`uint240`|  Message nonce to encode into.|
|`_version`|`uint16`|Version number to encode into the message nonce.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Message nonce with version encoded into the first two bytes.|


### decodeVersionedNonce

Pulls the version out of a version-encoded nonce.


```solidity
function decodeVersionedNonce(uint256 _nonce) internal pure returns (uint240, uint16);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_nonce`|`uint256`|Message nonce with version encoded into the first two bytes.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint240`|Nonce without encoded version.|
|`<none>`|`uint16`|Version of the message.|


