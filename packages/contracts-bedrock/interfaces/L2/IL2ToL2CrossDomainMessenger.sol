// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title IL2ToL2CrossDomainMessenger
/// @notice Interface for the L2ToL2CrossDomainMessenger contract.
interface IL2ToL2CrossDomainMessenger {
    /// @notice Thrown when a non-written slot in transient storage is attempted to be read from.
    error NotEntered();

    /// @notice Thrown when attempting to relay a message where payload origin is not L2ToL2CrossDomainMessenger.
    error IdOriginNotL2ToL2CrossDomainMessenger();

    /// @notice Thrown when the payload provided to the relay is not a SentMessage event.
    error EventPayloadNotSentMessage();

    /// @notice Thrown when attempting to send a message to the chain that the message is being sent from.
    error MessageDestinationSameChain();

    /// @notice Thrown when attempting to relay a message whose destination chain is not the chain relaying it.
    error MessageDestinationNotRelayChain();

    /// @notice Thrown when attempting to relay a message whose target is L2ToL2CrossDomainMessenger.
    error MessageTargetL2ToL2CrossDomainMessenger();

    /// @notice Thrown when attempting to relay a message that has already been relayed.
    error MessageAlreadyRelayed();

    /// @notice Thrown when a reentrant call is detected.
    error ReentrantCall();

    /// @notice Thrown when the provided message parameters do not match any hash of a previously sent message.
    error InvalidMessage();

    /// @notice Thrown when attempting to send or relay a message whose target is the
    ///         L2CrossDomainMessenger or the L2ToL1MessagePasser.
    error L2ToL2CrossDomainMessenger_MessageTargetUnsafe();

    /// @notice Thrown when a message is marked expired by anything but this chain's
    ///         L1CrossDomainMessenger.
    error L2ToL2CrossDomainMessenger_NotOtherMessenger();

    /// @notice Thrown when a message is marked expired on a fact that does not show it unrelayed
    ///         past the expiry period.
    error L2ToL2CrossDomainMessenger_MessageNotExpired();

    /// @notice Emitted whenever a message is sent to a destination
    /// @param destination  Chain ID of the destination chain.
    /// @param target       Target contract or wallet address.
    /// @param messageNonce Nonce associated with the message sent
    /// @param sender       Address initiating this message call
    /// @param message      Message payload to call target with.
    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    /// @notice Emitted whenever a message is successfully relayed on this chain.
    /// @param source       Chain ID of the source chain.
    /// @param messageNonce Nonce associated with the message sent
    /// @param messageHash  Hash of the message that was relayed.
    /// @param returnDataHash Hash of the return data from the message that was relayed.
    event RelayedMessage(
        uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash, bytes32 returnDataHash
    );

    /// @notice Emitted when a message sent from this chain is marked expired.
    /// @param messageHash   Hash of the message.
    /// @param undeliveredAt Destination timestamp at which the message had not been relayed.
    event MessageExpired(bytes32 indexed messageHash, uint256 undeliveredAt);

    function version() external view returns (string memory);

    /// @notice How long after it is sent a message must go unrelayed before it can be marked
    ///         expired.
    function EXPIRY_PERIOD() external view returns (uint256);

    /// @notice Mapping of message hashes to the timestamp of the block they were sent in.
    function sentMessageTimestamps(bytes32) external view returns (uint256);

    /// @notice Mapping of message hashes to whether they expired undelivered.
    function expiredMessages(bytes32) external view returns (bool);

    /// @notice Marks a message sent from this chain expired, on word from this chain's
    ///         L1CrossDomainMessenger.
    /// @param _messageHash   Hash of the message.
    /// @param _undeliveredAt Destination timestamp at which the message had not been relayed.
    function expireMessage(bytes32 _messageHash, uint256 _undeliveredAt) external;

    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only
    ///         be present in this mapping if it has successfully been relayed on this chain, and
    ///         can therefore not be relayed again.
    /// @return Returns true if the message corresponding to the `_msgHash` was successfully relayed.
    function successfulMessages(bytes32) external view returns (bool);

    /// @notice Retrieves the next message nonce. Message version will be added to the upper two
    ///         bytes of the message nonce. Message version allows us to treat messages as having
    ///         different structures.
    /// @return Nonce of the next message to be sent, with added message version.
    function messageNonce() external view returns (uint256);

    /// @notice Mapping of message nonces to message hashes. Note that a message will only be present in this
    ///         mapping if it has been sent from this chain to a destination chain.
    function sentMessages(uint256) external view returns (bytes32);

    /// @notice Retrieves the sender of the current cross domain message.
    /// @return sender_ Address of the sender of the current cross domain message.
    function crossDomainMessageSender() external view returns (address sender_);

    /// @notice Retrieves the source of the current cross domain message.
    /// @return source_ Chain ID of the source of the current cross domain message.
    function crossDomainMessageSource() external view returns (uint256 source_);

    /// @notice Retrieves the context of the current cross domain message. If not entered, reverts.
    /// @return sender_ Address of the sender of the current cross domain message.
    /// @return source_ Chain ID of the source of the current cross domain message.
    function crossDomainMessageContext() external view returns (address sender_, uint256 source_);

    /// @notice Sends a message to some target address on a destination chain. The destination chain
    ///         must differ from the current chain. The target cannot be the
    ///         L2ToL2CrossDomainMessenger, the L2CrossDomainMessenger or the L2ToL1MessagePasser. This
    ///         function is not payable, so no ETH can be sent with the message. If the relayed call
    ///         reverts, the message can be relayed again until it expires.
    /// @param _destination Chain ID of the destination chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message to trigger the target address with.
    /// @return messageHash_ The hash of the message being sent, used to track whether the message
    ///                      has successfully been relayed.
    function sendMessage(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 messageHash_);

    /// @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only
    ///         be executed via cross-chain call from the other messenger OR if the message was
    ///         already received once and is currently being replayed.
    /// @dev    If the target has no code when relayed (e.g. an EOA or a not-yet-deployed address),
    ///         the call is a no-op that still succeeds, permanently consuming the message; it cannot
    ///         be relayed again once the target is later deployed.
    /// @param _id          Identifier of the SentMessage event to be relayed
    /// @param _sentMessage Message payload of the `SentMessage` event
    /// @return returnData_ Return data from the target contract call.
    function relayMessage(
        Identifier calldata _id,
        bytes calldata _sentMessage
    )
        external
        payable
        returns (bytes memory returnData_);

    function messageVersion() external view returns (uint16);

    function __constructor__() external;
}
