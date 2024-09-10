// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { ISemver } from "src/universal/interfaces/ISemver.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { TransientReentrancyAware } from "src/libraries/TransientContext.sol";

/// @notice Thrown when a non-written slot in transient storage is attempted to be read from.
error NotEntered();

/// @notice Thrown when attempting to send a message to the chain that the message is being sent from.
error MessageDestinationSameChain();

/// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not CrossL2Inbox.
error RelayMessageCallerNotCrossL2Inbox();

/// @notice Thrown when attempting to relay a message where CrossL2Inbox's origin is not L2ToL2CrossDomainMessenger.
error CrossL2InboxOriginNotL2ToL2CrossDomainMessenger();

/// @notice Thrown when attempting to relay a message whose destination chain is not the chain relaying it.
error MessageDestinationNotRelayChain();

/// @notice Thrown when attempting to relay a message whose target is CrossL2Inbox.
error MessageTargetCrossL2Inbox();

/// @notice Thrown when attempting to relay a message whose target is L2ToL2CrossDomainMessenger.
error MessageTargetL2ToL2CrossDomainMessenger();

/// @notice Thrown when attempting to relay a message that has already been relayed.
error MessageAlreadyRelayed();

/// @notice Thrown when a reentrant call is detected.
error ReentrantCall();

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000023
/// @title L2ToL2CrossDomainMessenger
/// @notice The L2ToL2CrossDomainMessenger is a higher level abstraction on top of the CrossL2Inbox that provides
///         features necessary for secure transfers ERC20 tokens between L2 chains. Messages sent through the
///         L2ToL2CrossDomainMessenger on the source chain receive both replay protection as well as domain binding.
contract L2ToL2CrossDomainMessenger is IL2ToL2CrossDomainMessenger, ISemver, TransientReentrancyAware {
    /// @notice Storage slot for the sender of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.sender")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_SENDER_SLOT =
        0xb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f3;

    /// @notice Storage slot for the source of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.source")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_SOURCE_SLOT =
        0x711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee7;

    /// @notice Current message version identifier.
    uint16 public constant messageVersion = uint16(0);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.4
    string public constant version = "1.0.0-beta.4";

    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only be present in this
    ///         mapping if it has successfully been relayed on this chain, and can therefore not be relayed again.
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Nonce for the next message to be sent, without the message version applied. Use the messageNonce getter,
    ///         which will insert the message version into the nonce to give you the actual nonce to be used for the
    ///         message.
    uint240 internal msgNonce;

    /// @notice Emitted whenever a message is successfully relayed on this chain.
    /// @param messageHash Hash of the message that was relayed.
    event RelayedMessage(bytes32 indexed messageHash);

    /// @notice Emitted whenever a message fails to be relayed on this chain.
    /// @param messageHash Hash of the message that failed to be relayed.
    event FailedRelayedMessage(bytes32 indexed messageHash);

    /// @notice Retrieves the sender of the current cross domain message. If not entered, reverts.
    /// @return _sender Address of the sender of the current cross domain message.
    function crossDomainMessageSender() external view onlyEntered returns (address _sender) {
        assembly {
            _sender := tload(CROSS_DOMAIN_MESSAGE_SENDER_SLOT)
        }
    }

    /// @notice Retrieves the source of the current cross domain message. If not entered, reverts.
    /// @return _source Chain ID of the source of the current cross domain message.
    function crossDomainMessageSource() external view onlyEntered returns (uint256 _source) {
        assembly {
            _source := tload(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT)
        }
    }

    /// @notice Sends a message to some target address on a destination chain. Note that if the call always reverts,
    ///         then the message will be unrelayable and any ETH sent will be permanently locked. The same will occur
    ///         if the target on the other chain is considered unsafe (see the _isUnsafeTarget() function).
    /// @param _destination Chain ID of the destination chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message payload to call target with.
    function sendMessage(uint256 _destination, address _target, bytes calldata _message) external {
        if (_destination == block.chainid) revert MessageDestinationSameChain();
        if (_target == Predeploys.CROSS_L2_INBOX) revert MessageTargetCrossL2Inbox();
        if (_target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert MessageTargetL2ToL2CrossDomainMessenger();

        bytes memory data = abi.encodeCall(
            L2ToL2CrossDomainMessenger.relayMessage,
            (_destination, block.chainid, messageNonce(), msg.sender, _target, _message)
        );
        assembly {
            log0(add(data, 0x20), mload(data))
        }
        msgNonce++;
    }

    /// @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only be executed via
    ///         cross-chain call from the other messenger OR if the message was already received once and is currently
    ///         being replayed.
    /// @param _destination Chain ID of the destination chain.
    /// @param _source      Chain ID of the source chain.
    /// @param _nonce       Nonce of the message being relayed.
    /// @param _sender      Address of the user who sent the message.
    /// @param _target      Address that the message is targeted at.
    /// @param _message     Message payload to call target with.
    function relayMessage(
        uint256 _destination,
        uint256 _source,
        uint256 _nonce,
        address _sender,
        address _target,
        bytes memory _message
    )
        external
        payable
        nonReentrant
    {
        if (msg.sender != Predeploys.CROSS_L2_INBOX) revert RelayMessageCallerNotCrossL2Inbox();
        if (CrossL2Inbox(Predeploys.CROSS_L2_INBOX).origin() != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            revert CrossL2InboxOriginNotL2ToL2CrossDomainMessenger();
        }
        if (_destination != block.chainid) revert MessageDestinationNotRelayChain();
        if (_target == Predeploys.CROSS_L2_INBOX) revert MessageTargetCrossL2Inbox();
        if (_target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            revert MessageTargetL2ToL2CrossDomainMessenger();
        }

        bytes32 messageHash = keccak256(abi.encode(_destination, _source, _nonce, _sender, _target, _message));
        if (successfulMessages[messageHash]) {
            revert MessageAlreadyRelayed();
        }

        _storeMessageMetadata(_source, _sender);

        bool success = SafeCall.call(_target, msg.value, _message);

        if (success) {
            successfulMessages[messageHash] = true;
            emit RelayedMessage(messageHash);
        } else {
            emit FailedRelayedMessage(messageHash);
        }

        _storeMessageMetadata(0, address(0));
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
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)
            tstore(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT, _source)
        }
    }
}
