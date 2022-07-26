# L1CrossDomainMessenger



> L1CrossDomainMessenger



*The L1 Cross Domain Messenger contract sends messages from L1 to L2, and relays messages from L2 onto L1. In the event that a message sent from L1 to L2 is rejected for exceeding the L2 epoch gas limit, it can be resubmitted via this contract&#39;s replay function.*

## Methods

### allowMessage

```solidity
function allowMessage(bytes32 _xDomainCalldataHash) external nonpayable
```

Allow a message.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _xDomainCalldataHash | bytes32 | Hash of the message to block.

### blockMessage

```solidity
function blockMessage(bytes32 _xDomainCalldataHash) external nonpayable
```

Block a message.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _xDomainCalldataHash | bytes32 | Hash of the message to block.

### blockedMessages

```solidity
function blockedMessages(bytes32) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### initialize

```solidity
function initialize(address _libAddressManager) external nonpayable
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _libAddressManager | address | Address of the Address Manager.

### libAddressManager

```solidity
function libAddressManager() external view returns (contract Lib_AddressManager)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | contract Lib_AddressManager | undefined

### owner

```solidity
function owner() external view returns (address)
```



*Returns the address of the current owner.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### pause

```solidity
function pause() external nonpayable
```

Pause relaying.




### paused

```solidity
function paused() external view returns (bool)
```



*Returns true if the contract is paused, and false otherwise.*


#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### relayMessage

```solidity
function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, IL1CrossDomainMessenger.L2MessageInclusionProof _proof) external nonpayable
```

Relays a cross domain message to a contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _target | address | Target contract address.
| _sender | address | Message sender address.
| _message | bytes | Message to send to the target.
| _messageNonce | uint256 | Nonce for the provided message.
| _proof | IL1CrossDomainMessenger.L2MessageInclusionProof | Inclusion proof for the given message.

### relayedMessages

```solidity
function relayedMessages(bytes32) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### renounceOwnership

```solidity
function renounceOwnership() external nonpayable
```



*Leaves the contract without owner. It will not be possible to call `onlyOwner` functions anymore. Can only be called by the current owner. NOTE: Renouncing ownership will leave the contract without an owner, thereby removing any functionality that is only available to the owner.*


### replayMessage

```solidity
function replayMessage(address _target, address _sender, bytes _message, uint256 _queueIndex, uint32 _oldGasLimit, uint32 _newGasLimit) external nonpayable
```

Replays a cross domain message to the target messenger.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _target | address | Target contract address.
| _sender | address | Original sender address.
| _message | bytes | Message to send to the target.
| _queueIndex | uint256 | CTC Queue index for the message to replay.
| _oldGasLimit | uint32 | Original gas limit used to send the message.
| _newGasLimit | uint32 | New gas limit to be used for this message.

### resolve

```solidity
function resolve(string _name) external view returns (address)
```

Resolves the address associated with a given name.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _name | string | Name to resolve an address for.

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | Address associated with the given name.

### sendMessage

```solidity
function sendMessage(address _target, bytes _message, uint32 _gasLimit) external nonpayable
```

Sends a cross domain message to the target messenger.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _target | address | Target contract address.
| _message | bytes | Message to send to the target.
| _gasLimit | uint32 | Gas limit for the provided message.

### successfulMessages

```solidity
function successfulMessages(bytes32) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

### transferOwnership

```solidity
function transferOwnership(address newOwner) external nonpayable
```



*Transfers ownership of the contract to a new account (`newOwner`). Can only be called by the current owner.*

#### Parameters

| Name | Type | Description |
|---|---|---|
| newOwner | address | undefined

### xDomainMessageSender

```solidity
function xDomainMessageSender() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined



## Events

### FailedRelayedMessage

```solidity
event FailedRelayedMessage(bytes32 indexed msgHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| msgHash `indexed` | bytes32 | undefined |

### Initialized

```solidity
event Initialized(uint8 version)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| version  | uint8 | undefined |

### MessageAllowed

```solidity
event MessageAllowed(bytes32 indexed _xDomainCalldataHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _xDomainCalldataHash `indexed` | bytes32 | undefined |

### MessageBlocked

```solidity
event MessageBlocked(bytes32 indexed _xDomainCalldataHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _xDomainCalldataHash `indexed` | bytes32 | undefined |

### OwnershipTransferred

```solidity
event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| previousOwner `indexed` | address | undefined |
| newOwner `indexed` | address | undefined |

### Paused

```solidity
event Paused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |

### RelayedMessage

```solidity
event RelayedMessage(bytes32 indexed msgHash)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| msgHash `indexed` | bytes32 | undefined |

### SentMessage

```solidity
event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| target `indexed` | address | undefined |
| sender  | address | undefined |
| message  | bytes | undefined |
| messageNonce  | uint256 | undefined |
| gasLimit  | uint256 | undefined |

### Unpaused

```solidity
event Unpaused(address account)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| account  | address | undefined |



