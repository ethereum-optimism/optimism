# Cross Domain Messengers

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**

- [Message Passing](#message-passing)
- [Upgradability](#upgradability)
- [Message Versioning](#message-versioning)
  - [Message Version 0](#message-version-0)
  - [Message Version 1](#message-version-1)
- [Backwards Compatibility Notes](#backwards-compatibility-notes)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

The cross domain messengers are responsible for providing a higher level API for
developers who are interested in sending cross domain messages. They allow for
the ability to replay cross domain messages and sit directly on top of the lower
level system contracts responsible for cross domain messaging on L1 and L2.

The `CrossDomainMessenger` is extended to create both an
`L1CrossDomainMessenger` and well as a `L2CrossDomainMessenger`.
These contracts are then extended with their legacy APIs to provide backwards
compatibility for applications that integrated before the Bedrock system
upgrade.

The `L2CrossDomainMessenger` is a predeploy contract located at
`0x4200000000000000000000000000000000000007`.

The base `CrossDomainMessenger` interface is:

```solidity
interface CrossDomainMessenger {
    event FailedRelayedMessage(bytes32 indexed msgHash);
    event RelayedMessage(bytes32 indexed msgHash);
    event SentMessage(address indexed target, address sender, bytes message, uint256 messageNonce, uint256 gasLimit);
    event SentMessageExtension1(address indexed sender, uint256 value);

    function MESSAGE_VERSION() external view returns (uint16);
    function MIN_GAS_CALLDATA_OVERHEAD() external view returns (uint64);
    function MIN_GAS_CONSTANT_OVERHEAD() external view returns (uint64);
    function MIN_GAS_DYNAMIC_OVERHEAD_DENOMINATOR() external view returns (uint64);
    function MIN_GAS_DYNAMIC_OVERHEAD_NUMERATOR() external view returns (uint64);
    function OTHER_MESSENGER() external view returns (address);
    function baseGas(bytes memory _message, uint32 _minGasLimit) external pure returns (uint64);
    function failedMessages(bytes32) external view returns (bool);
    function messageNonce() external view returns (uint256);
    function relayMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes memory _message
    ) external payable;
    function sendMessage(address _target, bytes memory _message, uint32 _minGasLimit) external payable;
    function successfulMessages(bytes32) external view returns (bool);
    function xDomainMessageSender() external view returns (address);
}
```

## Message Passing

The `sendMessage` function is used to send a cross domain message. To trigger
the execution on the other side, the `relayMessage` function is called.
Successful messages have their hash stored in the `successfulMessages` mapping
while unsuccessful messages have their hash stored in the `failedMessages`
mapping.

The user experience when sending from L1 to L2 is a bit different than when
sending a transaction from L2 to L1. When going into L1 from L2, the user does
not need to call `relayMessage` on L2 themselves. The user pays for L2 gas on L1
and the transaction is automatically pulled into L2 where it is executed on L2.
When going from L2 into L1, the user proves their withdrawal on OptimismPortal,
then waits for the finalization window to pass, and then finalizes the withdrawal
on the OptimismPortal, which calls `relayMessage` on the
`L1CrossDomainMessenger` to finalize the withdrawal.

## Upgradability

The L1 and L2 cross domain messengers should be deployed behind upgradable
proxies. This will allow for updating the message version.

## Message Versioning

Messages are versioned based on the first 2 bytes of their nonce. Depending on
the version, messages can have a different serialization and hashing scheme.
The first two bytes of the nonce are reserved for version metadata because
a version field was not originally included in the messages themselves, but
a `uint256` nonce is so large that we can very easily pack additional data
into that field.

### Message Version 0

```solidity
abi.encodeWithSignature(
    "relayMessage(address,address,bytes,uint256)",
    _target,
    _sender,
    _message,
    _messageNonce
);
```

### Message Version 1

```solidity
abi.encodeWithSignature(
    "relayMessage(uint256,address,address,uint256,uint256,bytes)",
    _nonce,
    _sender,
    _target,
    _value,
    _gasLimit,
    _data
);
```

## Backwards Compatibility Notes

An older version of the messenger contracts had the concept of blocked messages
in a `blockedMessages` mapping. This functionality was removed from the
messengers because a smart attacker could get around any message blocking
attempts. It also saves gas on finalizing withdrawals.

The concept of a "relay id" and the `relayedMessages` mapping was removed.
It was built as a way to be able to fund third parties who relayed messages
on the behalf of users, but it was improperly implemented as it was impossible
to know if the relayed message actually succeeded.
