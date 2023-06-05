# CrossDomainMessenger
[Git Source](https://github.com/ethereum-optimism/optimism/blob/f7b73857601914eeea6fc4c1ba46ae99ca744d97/contracts/universal/CrossDomainMessenger.sol)

**Inherits:**
[CrossDomainMessengerLegacySpacer0](/contracts/universal/CrossDomainMessenger.sol/contract.CrossDomainMessengerLegacySpacer0.md), Initializable, [CrossDomainMessengerLegacySpacer1](/contracts/universal/CrossDomainMessenger.sol/contract.CrossDomainMessengerLegacySpacer1.md)

CrossDomainMessenger is a base contract that provides the core logic for the L1 and L2
cross-chain messenger contracts. It's designed to be a universal interface that only
needs to be extended slightly to provide low-level message passing functionality on each
chain it's deployed on. Currently only designed for message passing between two paired
chains and does not support one-to-many interactions.
Any changes to this contract MUST result in a semver bump for contracts that inherit it.


## State Variables
### MESSAGE_VERSION
Current message version identifier.


```solidity
uint16 public constant MESSAGE_VERSION = 1;
```


### RELAY_CONSTANT_OVERHEAD
Constant overhead added to the base gas for a message.


```solidity
uint64 public constant RELAY_CONSTANT_OVERHEAD = 200_000;
```


### MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR
Numerator for dynamic overhead added to the base gas for a message.


```solidity
uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR = 64;
```


### MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR
Denominator for dynamic overhead added to the base gas for a message.


```solidity
uint64 public constant MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR = 63;
```


### MIN_GAS_CALLDATA_OVERHEAD
Extra gas added to base gas for each byte of calldata in a message.


```solidity
uint64 public constant MIN_GAS_CALLDATA_OVERHEAD = 16;
```


### RELAY_CALL_OVERHEAD
Gas reserved for performing the external call in `relayMessage`.


```solidity
uint64 public constant RELAY_CALL_OVERHEAD = 40_000;
```


### RELAY_RESERVED_GAS
Gas reserved for finalizing the execution of `relayMessage` after the safe call.


```solidity
uint64 public constant RELAY_RESERVED_GAS = 40_000;
```


### RELAY_GAS_CHECK_BUFFER
Gas reserved for the execution between the `hasMinGas` check and the external
call in `relayMessage`.


```solidity
uint64 public constant RELAY_GAS_CHECK_BUFFER = 5_000;
```


### OTHER_MESSENGER
Address of the paired CrossDomainMessenger contract on the other chain.


```solidity
address public immutable OTHER_MESSENGER;
```


### successfulMessages
Mapping of message hashes to boolean receipt values. Note that a message will only
be present in this mapping if it has successfully been relayed on this chain, and
can therefore not be relayed again.


```solidity
mapping(bytes32 => bool) public successfulMessages;
```


### xDomainMsgSender
Address of the sender of the currently executing message on the other chain. If the
value of this variable is the default value (0x00000000...dead) then no message is
currently being executed. Use the xDomainMessageSender getter which will throw an
error if this is the case.


```solidity
address internal xDomainMsgSender;
```


### msgNonce
Nonce for the next message to be sent, without the message version applied. Use the
messageNonce getter which will insert the message version into the nonce to give you
the actual nonce to be used for the message.


```solidity
uint240 internal msgNonce;
```


### failedMessages
Mapping of message hashes to a boolean if and only if the message has failed to be
executed at least once. A message will not be present in this mapping if it
successfully executed on the first attempt.


```solidity
mapping(bytes32 => bool) public failedMessages;
```


### __gap
Reserve extra slots in the storage layout for future upgrades.
A gap size of 41 was chosen here, so that the first slot used in a child contract
would be a multiple of 50.


```solidity
uint256[42] private __gap;
```


## Functions
### constructor


```solidity
constructor(address _otherMessenger);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_otherMessenger`|`address`|Address of the messenger on the paired chain.|


### sendMessage

Sends a message to some target address on the other chain. Note that if the call
always reverts, then the message will be unrelayable, and any ETH sent will be
permanently locked. The same will occur if the target on the other chain is
considered unsafe (see the _isUnsafeTarget() function).


```solidity
function sendMessage(address _target, bytes calldata _message, uint32 _minGasLimit) external payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|     Target contract or wallet address.|
|`_message`|`bytes`|    Message to trigger the target address with.|
|`_minGasLimit`|`uint32`|Minimum gas limit that the message can be executed with.|


### relayMessage

Relays a message that was sent by the other CrossDomainMessenger contract. Can only
be executed via cross-chain call from the other messenger OR if the message was
already received once and is currently being replayed.


```solidity
function relayMessage(
    uint256 _nonce,
    address _sender,
    address _target,
    uint256 _value,
    uint256 _minGasLimit,
    bytes calldata _message
) external payable;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_nonce`|`uint256`|      Nonce of the message being relayed.|
|`_sender`|`address`|     Address of the user who sent the message.|
|`_target`|`address`|     Address that the message is targeted at.|
|`_value`|`uint256`|      ETH value to send with the message.|
|`_minGasLimit`|`uint256`|Minimum amount of gas that the message can be executed with.|
|`_message`|`bytes`|    Message to send to the target.|


### xDomainMessageSender

Retrieves the address of the contract or wallet that initiated the currently
executing message on the other chain. Will throw an error if there is no message
currently being executed. Allows the recipient of a call to see who triggered it.


```solidity
function xDomainMessageSender() external view returns (address);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`address`|Address of the sender of the currently executing message on the other chain.|


### messageNonce

Retrieves the next message nonce. Message version will be added to the upper two
bytes of the message nonce. Message version allows us to treat messages as having
different structures.


```solidity
function messageNonce() public view returns (uint256);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint256`|Nonce of the next message to be sent, with added message version.|


### baseGas

Computes the amount of gas required to guarantee that a given message will be
received on the other chain without running out of gas. Guaranteeing that a message
will not run out of gas is important because this ensures that a message can always
be replayed on the other chain if it fails to execute completely.


```solidity
function baseGas(bytes calldata _message, uint32 _minGasLimit) public pure returns (uint64);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_message`|`bytes`|    Message to compute the amount of required gas for.|
|`_minGasLimit`|`uint32`|Minimum desired gas limit when message goes to target.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`uint64`|Amount of gas required to guarantee message receipt.|


### __CrossDomainMessenger_init

Intializer.


```solidity
function __CrossDomainMessenger_init() internal onlyInitializing;
```

### _sendMessage

Sends a low-level message to the other messenger. Needs to be implemented by child
contracts because the logic for this depends on the network where the messenger is
being deployed.


```solidity
function _sendMessage(address _to, uint64 _gasLimit, uint256 _value, bytes memory _data) internal virtual;
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_to`|`address`|      Recipient of the message on the other chain.|
|`_gasLimit`|`uint64`|Minimum gas limit the message can be executed with.|
|`_value`|`uint256`|   Amount of ETH to send with the message.|
|`_data`|`bytes`|    Message data.|


### _isOtherMessenger

Checks whether the message is coming from the other messenger. Implemented by child
contracts because the logic for this depends on the network where the messenger is
being deployed.


```solidity
function _isOtherMessenger() internal view virtual returns (bool);
```
**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether the message is coming from the other messenger.|


### _isUnsafeTarget

Checks whether a given call target is a system address that could cause the
messenger to peform an unsafe action. This is NOT a mechanism for blocking user
addresses. This is ONLY used to prevent the execution of messages to specific
system addresses that could cause security issues, e.g., having the
CrossDomainMessenger send messages to itself.


```solidity
function _isUnsafeTarget(address _target) internal view virtual returns (bool);
```
**Parameters**

|Name|Type|Description|
|----|----|-----------|
|`_target`|`address`|Address of the contract to check.|

**Returns**

|Name|Type|Description|
|----|----|-----------|
|`<none>`|`bool`|Whether or not the address is an unsafe system address.|


## Events
### SentMessage
Emitted whenever a message is sent to the other chain.


```solidity
event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit);
```

### SentMessageExtension1
Additional event data to emit, required as of Bedrock. Cannot be merged with the
SentMessage event without breaking the ABI of this contract, this is good enough.


```solidity
event SentMessageExtension1(address indexed sender, uint256 value);
```

### RelayedMessage
Emitted whenever a message is successfully relayed on this chain.


```solidity
event RelayedMessage(bytes32 indexed msgHash);
```

### FailedRelayedMessage
Emitted whenever a message fails to be relayed on this chain.


```solidity
event FailedRelayedMessage(bytes32 indexed msgHash);
```

