# L2CrossDomainMessenger



> L2CrossDomainMessenger



*The L2 Cross Domain Messenger contract sends messages from L2 to L1, and is the entry point for L2 messages sent via the L1 Cross Domain Messenger.*

## Methods

### l1CrossDomainMessenger

```solidity
function l1CrossDomainMessenger() external view returns (address)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | address | undefined

### messageNonce

```solidity
function messageNonce() external view returns (uint256)
```






#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | uint256 | undefined

### relayMessage

```solidity
function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce) external nonpayable
```

Relays a cross domain message to a contract.



#### Parameters

| Name | Type | Description |
|---|---|---|
| _target | address | Target contract address.
| _sender | address | Message sender address.
| _message | bytes | Message to send to the target.
| _messageNonce | uint256 | Nonce for the provided message.

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

### sentMessages

```solidity
function sentMessages(bytes32) external view returns (bool)
```





#### Parameters

| Name | Type | Description |
|---|---|---|
| _0 | bytes32 | undefined

#### Returns

| Name | Type | Description |
|---|---|---|
| _0 | bool | undefined

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



