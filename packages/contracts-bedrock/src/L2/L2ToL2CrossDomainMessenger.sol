// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Encoding } from "src/libraries/Encoding.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { TransientReentrancyAware } from "src/libraries/TransientContext.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";

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

/// @notice Thrown when the provided message parameters do not match any hash of a previously sent message.
error InvalidMessage();

/// @notice Thrown when attempting to send or relay a message whose target is the
///         L2CrossDomainMessenger or the L2ToL1MessagePasser.
error L2ToL2CrossDomainMessenger_MessageTargetUnsafe();

/// @notice Thrown when a message is marked expired by anything but this chain's
///         L1CrossDomainMessenger.
error L2ToL2CrossDomainMessenger_NotOtherMessenger();

/// @notice Thrown when a message is marked expired on a fact that does not show it unrelayed past
///         the expiry period.
error L2ToL2CrossDomainMessenger_MessageNotExpired();

/// @notice Thrown when the contract is deployed with a zero expiry period.
error L2ToL2CrossDomainMessenger_InvalidExpiryPeriod();

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000023
/// @title L2ToL2CrossDomainMessenger
/// @notice The L2ToL2CrossDomainMessenger is a higher level abstraction on top of the CrossL2Inbox that provides
///         features necessary for secure transfers ERC20 tokens between L2 chains. Messages sent through the
///         L2ToL2CrossDomainMessenger on the source chain receive both replay protection as well as domain binding.
contract L2ToL2CrossDomainMessenger is ISemver, TransientReentrancyAware {
    /// @notice Storage slot for the sender of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.sender")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_SENDER_SLOT =
        0xb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f3;

    /// @notice Storage slot for the source of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.source")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_SOURCE_SLOT =
        0x711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee7;

    /// @notice Event selector for the SentMessage event. Will be removed in favor of reading
    //          the `selector` property directly once crytic/slithe/#2566 is fixed.
    bytes32 internal constant SENT_MESSAGE_EVENT_SELECTOR =
        0x382409ac69001e11931a28435afef442cbfd20d9891907e8fa373ba7d351f320;

    /// @notice Current message version identifier.
    uint16 public constant messageVersion = uint16(0);

    /// @notice How long after it is sent a message must go unrelayed before it can be marked
    ///         expired. The protocol rejects an executing message whose block is more than the
    ///         message expiry window after the block of its initiating message, and the interop
    ///         specification caps that window at 7 days. The period must exceed the window, so a
    ///         message is only marked expired once no relay of it can be valid. Production
    ///         deployments use 8 days: the window plus a day of margin. It is set at deployment so
    ///         that test networks with a shorter window can use a shorter period.
    uint256 internal immutable EXPIRY_PERIOD;

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only be present in this
    ///         mapping if it has successfully been relayed on this chain, and can therefore not be relayed again.
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Nonce for the next message to be sent, without the message version applied. Use the messageNonce getter,
    ///         which will insert the message version into the nonce to give you the actual nonce to be used for the
    ///         message.
    uint240 internal msgNonce;

    /// @notice Mapping of message nonces to message hashes. Note that a message will only be present in this
    ///         mapping if it has been sent from this chain to a destination chain.
    mapping(uint256 => bytes32) public sentMessages;

    /// @notice Mapping of message hashes to the timestamp of the block they were sent in. Note that
    ///         a message will only be present in this mapping if it has been sent from this chain
    ///         to a destination chain. Messages sent before this mapping existed are absent and can
    ///         never expire.
    mapping(bytes32 => uint256) public sentMessageTimestamps;

    /// @notice Mapping of message hashes to whether they expired. A message sent from this chain
    ///         expires when its destination shows it was not relayed by the end of the expiry
    ///         period, after which it never can be. Applications read this to undo a send.
    mapping(bytes32 => bool) public expiredMessages;

    /// @notice Emitted whenever a message is sent to a destination
    /// @param destination  Chain ID of the destination chain.
    /// @param target       Target contract or wallet address.
    /// @param messageNonce Nonce associated with the message sent
    /// @param sender       Address initiating this message call
    /// @param message      Message payload to call target with.
    event SentMessage(
        uint256 indexed destination, address indexed target, uint256 indexed messageNonce, address sender, bytes message
    );

    /// @notice Emitted when a message sent from this chain is marked expired.
    /// @param messageHash   Hash of the message.
    /// @param undeliveredAt Destination timestamp at which the message had not been relayed.
    event MessageExpired(bytes32 indexed messageHash, uint256 undeliveredAt);

    /// @notice Emitted whenever a message is successfully relayed on this chain.
    /// @param source       Chain ID of the source chain.
    /// @param messageNonce Nonce associated with the message sent
    /// @param messageHash  Hash of the message that was relayed.
    /// @param returnDataHash Hash of the return data from the message that was relayed.
    event RelayedMessage(
        uint256 indexed source, uint256 indexed messageNonce, bytes32 indexed messageHash, bytes32 returnDataHash
    );

    /// @notice Constructs the L2ToL2CrossDomainMessenger.
    /// @param _expiryPeriod How long after it is sent a message must go unrelayed before it can
    ///                      be marked expired. See `EXPIRY_PERIOD`.
    constructor(uint256 _expiryPeriod) {
        if (_expiryPeriod == 0) revert L2ToL2CrossDomainMessenger_InvalidExpiryPeriod();
        EXPIRY_PERIOD = _expiryPeriod;
    }

    /// @notice Returns how long after it is sent a message must go unrelayed before it can be
    ///         marked expired. See `EXPIRY_PERIOD`.
    /// @return The expiry period, in seconds.
    function expiryPeriod() public view returns (uint256) {
        return EXPIRY_PERIOD;
    }

    /// @notice Retrieves the sender of the current cross domain message. If not entered, reverts.
    /// @return sender_ Address of the sender of the current cross domain message.
    function crossDomainMessageSender() external view onlyEntered returns (address sender_) {
        assembly {
            sender_ := tload(CROSS_DOMAIN_MESSAGE_SENDER_SLOT)
        }
    }

    /// @notice Retrieves the source of the current cross domain message. If not entered, reverts.
    /// @return source_ Chain ID of the source of the current cross domain message.
    function crossDomainMessageSource() external view onlyEntered returns (uint256 source_) {
        assembly {
            source_ := tload(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT)
        }
    }

    /// @notice Retrieves the context of the current cross domain message. If not entered, reverts.
    /// @return sender_ Address of the sender of the current cross domain message.
    /// @return source_ Chain ID of the source of the current cross domain message.
    function crossDomainMessageContext() external view onlyEntered returns (address sender_, uint256 source_) {
        assembly {
            sender_ := tload(CROSS_DOMAIN_MESSAGE_SENDER_SLOT)
            source_ := tload(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT)
        }
    }

    /// @notice Sends a message to some target address on a destination chain. The destination
    ///         chain must differ from the current chain. The target cannot be the
    ///         L2ToL2CrossDomainMessenger, the L2CrossDomainMessenger or the
    ///         L2ToL1MessagePasser. This function is not payable, so no ETH can be sent with the
    ///         message. If the relayed call reverts, the message can be relayed again until it
    ///         expires.
    /// @param _destination Chain ID of the destination chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message payload to call target with.
    /// @return messageHash_ The hash of the message being sent, used to track whether the message
    ///                      has successfully been relayed.
    function sendMessage(
        uint256 _destination,
        address _target,
        bytes calldata _message
    )
        external
        returns (bytes32 messageHash_)
    {
        if (_destination == block.chainid) revert MessageDestinationSameChain();
        if (_target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2ToL2CrossDomainMessenger();
        if (_isUnsafeTarget(_target)) revert L2ToL2CrossDomainMessenger_MessageTargetUnsafe();

        uint256 nonce = messageNonce();
        messageHash_ = Hashing.hashL2toL2CrossDomainMessage({
            _destination: _destination,
            _source: block.chainid,
            _nonce: nonce,
            _sender: msg.sender,
            _target: _target,
            _message: _message
        });

        sentMessages[nonce] = messageHash_;
        sentMessageTimestamps[messageHash_] = block.timestamp;
        msgNonce++;

        emit SentMessage(_destination, _target, nonce, msg.sender, _message);
    }

    /// @notice Relays a message that was sent by the other L2ToL2CrossDomainMessenger contract. Can only be executed
    ///         via cross chain call from the other messenger OR if the message was already received once and is
    ///         currently being replayed.
    /// @dev    If the target has no code when relayed (e.g. an EOA or a not-yet-deployed address), the call is a
    ///         no-op that still succeeds, permanently consuming the message; it cannot be relayed again once the
    ///         target is later deployed.
    /// @param _id          Identifier of the SentMessage event to be relayed
    /// @param _sentMessage Payload of the `SentMessage` event
    /// @return returnData_ Return data from the target contract call.
    function relayMessage(
        Identifier calldata _id,
        bytes calldata _sentMessage
    )
        external
        payable
        nonReentrant
        returns (bytes memory returnData_)
    {
        // Ensure the log came from the messenger.
        if (_id.origin != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            revert IdOriginNotL2ToL2CrossDomainMessenger();
        }

        // Signal that this is a cross chain call that needs to have the identifier validated
        ICrossL2Inbox(Predeploys.CROSS_L2_INBOX).validateMessage(_id, keccak256(_sentMessage));

        // Decode the payload
        (uint256 destination, address target, uint256 nonce, address sender, bytes memory message) =
            _decodeSentMessagePayload(_sentMessage);

        // Assert invariants on the message
        if (destination != block.chainid) revert MessageDestinationNotRelayChain();

        if (_isUnsafeTarget(target)) revert L2ToL2CrossDomainMessenger_MessageTargetUnsafe();

        uint256 source = _id.chainId;
        bytes32 messageHash = Hashing.hashL2toL2CrossDomainMessage({
            _destination: destination,
            _source: source,
            _nonce: nonce,
            _sender: sender,
            _target: target,
            _message: message
        });

        if (successfulMessages[messageHash]) {
            revert MessageAlreadyRelayed();
        }

        successfulMessages[messageHash] = true;
        _storeMessageMetadata(source, sender);

        bool success;
        (success, returnData_) = target.call{ value: msg.value }(message);

        if (!success) {
            assembly {
                revert(add(32, returnData_), mload(returnData_))
            }
        }

        emit RelayedMessage(source, nonce, messageHash, keccak256(returnData_));

        _storeMessageMetadata(0, address(0));
    }

    /// @notice Marks a message sent from this chain expired, on word from this chain's
    ///         L1CrossDomainMessenger that its destination had not relayed it by `_undeliveredAt`.
    ///         Only the destination can have produced that word, since it computed the message hash
    ///         with its own chain ID. Word for a message that already expired changes nothing and
    ///         emits nothing.
    /// @param _messageHash   Hash of the message.
    /// @param _undeliveredAt Destination timestamp at which the message had not been relayed.
    function expireMessage(bytes32 _messageHash, uint256 _undeliveredAt) external {
        if (
            msg.sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER
                || ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).xDomainMessageSender()
                    != address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger())
        ) {
            revert L2ToL2CrossDomainMessenger_NotOtherMessenger();
        }
        if (expiredMessages[_messageHash]) return;

        uint256 sentAt = sentMessageTimestamps[_messageHash];
        if (sentAt == 0) revert InvalidMessage();
        if (_undeliveredAt <= sentAt + EXPIRY_PERIOD) revert L2ToL2CrossDomainMessenger_MessageNotExpired();

        expiredMessages[_messageHash] = true;

        emit MessageExpired(_messageHash, _undeliveredAt);
    }

    /// @notice Checks whether a message may not target an address. No relayed message may call the
    ///         L2CrossDomainMessenger or the L2ToL1MessagePasser, so this contract never initiates
    ///         a withdrawal, and nothing that trusts it as a withdrawal's sender, today or in a
    ///         future L1 contract, can be fooled by a relayed message. Expiry does not rely on
    ///         this: L1CrossDomainMessengers trust a dedicated exporter predeploy as the sender of
    ///         undelivered-message word, not this contract.
    /// @param _target Target of the message.
    /// @return Whether the target is unsafe.
    function _isUnsafeTarget(address _target) internal pure returns (bool) {
        return _target == Predeploys.L2_CROSS_DOMAIN_MESSENGER || _target == Predeploys.L2_TO_L1_MESSAGE_PASSER;
    }

    /// @notice Retrieves the next message nonce. Message version will be added to the upper two bytes of the message
    ///         nonce. Message version allows us to treat messages as having different structures.
    /// @return Nonce of the next message to be sent, with added message version.
    function messageNonce() public view returns (uint256) {
        return Encoding.encodeVersionedNonce(msgNonce, messageVersion);
    }

    /// @notice Stores message data such as sender and source in transient storage.
    /// @param _source Chain ID of the source chain.
    /// @param _sender Address of the sender of the message.
    function _storeMessageMetadata(uint256 _source, address _sender) internal {
        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT, _source)
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)
        }
    }

    /// @notice Decodes the payload of a SentMessage event.
    /// @dev    The payload format is as follows:
    ///         encodePacked(
    ///               encode(event selector, destination, target, nonce),
    ///               encode(sender, message)
    ///         )
    /// @param _payload         Payload of the SentMessage event.
    /// @return destination_    Destination chain ID.
    /// @return target_         Target contract of the message.
    /// @return nonce_          Nonce associated with the message sent.
    /// @return sender_         Address initiating this message call.
    /// @return message_        Message payload to call target with.
    function _decodeSentMessagePayload(bytes calldata _payload)
        internal
        pure
        returns (uint256 destination_, address target_, uint256 nonce_, address sender_, bytes memory message_)
    {
        // Validate Selector (also reverts if LOG0 with no topics)
        bytes32 selector = abi.decode(_payload[:32], (bytes32));
        if (selector != SENT_MESSAGE_EVENT_SELECTOR) revert EventPayloadNotSentMessage();

        // Topics
        (destination_, target_, nonce_) = abi.decode(_payload[32:128], (uint256, address, uint256));

        // Data
        (sender_, message_) = abi.decode(_payload[128:], (address, bytes));
    }
}
