# IL1CrossDomainMessenger





> IL1CrossDomainMessenger









## Methods


### relayMessage


```solidity
function relayMessage(address _target, address _sender, bytes _message, uint256 _messageNonce, IL1CrossDomainMessenger.L2MessageInclusionProof _proof) external nonpayable

```








#### Parameters

| Name | Type | Description |
|---|---|---|

| _target | address | undefined


| _sender | address | undefined


| _message | bytes | undefined


| _messageNonce | uint256 | undefined


| _proof | IL1CrossDomainMessenger.L2MessageInclusionProof | undefined






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








