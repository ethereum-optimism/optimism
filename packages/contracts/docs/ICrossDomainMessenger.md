# ICrossDomainMessenger



> ICrossDomainMessenger





## Methods

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



