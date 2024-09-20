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

    /// @notice Storage slot controlling whether or not to batch cross domain messages.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.isbatching")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_IS_BATCHING_SLOT =
        0xaac43bf59350fd9eacb862eb14d57a59b0fbde3051d74d34d63145e0dcbc4ea8;

    /// @notice Storage slot for storing the length of batched cross domain messages.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.batchmessagecount")) - 1)
    bytes32 internal constant CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT =
        0x7b7a8507cffe44857faca837650e78002747b183d7a2fbb0c91586ae7b37deac;

    /// @notice Current message version identifier.
    uint16 public constant messageVersion = uint16(0);

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.5
    string public constant version = "1.0.0-beta.5";

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

    /// @notice Retrieves whether the current message is a batch.
    /// @return _isBatching true if the cross domain messenger is batching messages.
    function crossDomainMessageIsBatching() internal view returns (bool _isBatching) {
        assembly {
            _isBatching := tload(CROSS_DOMAIN_MESSAGE_IS_BATCHING_SLOT)
        }
    }

    /// @notice The sent messages will be emitted in a batch.
    function startBatching() external {
        _storeIsBatching(true);
    }

    /// @notice Emit all messages from current batch.
    function finishBatchingAndSend() external {
        if (L2ToL2CrossDomainMessenger.crossDomainMessageIsBatching() == false) revert();

        BatchableMessage[] memory allMessages = _processAllMessages();
        _sendBatchedMessage(allMessages);

        _storeIsBatching(false);
        _clearAllMessages();
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

        if (L2ToL2CrossDomainMessenger.crossDomainMessageIsBatching() == true) {
            BatchableMessage memory newMessage =
                BatchableMessage({ sender: msg.sender, target: _target, destination: _destination, message: _message });
            _appendBatchMessage(newMessage);
        } else {
            bytes memory data = abi.encodeCall(
                L2ToL2CrossDomainMessenger.relayMessage,
                (_destination, block.chainid, messageNonce(), msg.sender, _target, _message)
            );
            assembly {
                log0(add(data, 0x20), mload(data))
            }
            msgNonce++;
        }
    }

    /// @notice Sends a batched message to a destination chain. Note that if the call always reverts,
    ///         then the message will be unrelayable and any ETH sent will be permanently locked. The same will occur
    ///         if the target on the other chain is considered unsafe (see the _isUnsafeTarget() function).
    /// @param _batchedMessage      Batched message to send to destination chain.
    function _sendBatchedMessage(BatchableMessage[] memory _batchedMessage) internal {
        for (uint256 i = 0; i < _batchedMessage.length; i++) {
            BatchableMessage memory message = _batchedMessage[i];
            if (message.destination == block.chainid) revert MessageDestinationSameChain();
            if (message.target == Predeploys.CROSS_L2_INBOX) revert MessageTargetCrossL2Inbox();
            if (message.target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
                revert MessageTargetL2ToL2CrossDomainMessenger();
            }
        }

        bytes memory data = abi.encodeCall(this.relayBatchedMessage, (block.chainid, messageNonce(), _batchedMessage));

        assembly {
            log0(add(data, 0x20), mload(data))
        }

        msgNonce++;
    }

    /// @notice Relays a batched message that was sent by the other CrossDomainMessenger contract. Can only be executed
    /// via
    ///         cross-chain call from the other messenger OR if the message was already received once and is currently
    ///         being replayed.
    /// @param _source              Chain ID of the source chain.
    /// @param _nonce               Nonce of the message being relayed.
    /// @param _batchedMessage      Batched message to iterate over and call.
    function relayBatchedMessage(
        uint256 _source,
        uint256 _nonce,
        BatchableMessage[] memory _batchedMessage
    )
        external
        payable
    {
        if (msg.sender != Predeploys.CROSS_L2_INBOX) revert RelayMessageCallerNotCrossL2Inbox();
        if (CrossL2Inbox(Predeploys.CROSS_L2_INBOX).origin() != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            revert CrossL2InboxOriginNotL2ToL2CrossDomainMessenger();
        }

        for (uint256 i = 0; i < _batchedMessage.length; i++) {
            BatchableMessage memory message = _batchedMessage[i];
            if (message.destination != block.chainid) revert MessageDestinationNotRelayChain();
            if (message.target == Predeploys.CROSS_L2_INBOX) revert MessageTargetCrossL2Inbox();
            if (message.target == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
                revert MessageTargetL2ToL2CrossDomainMessenger();
            }

            bytes32 messageHash = keccak256(
                abi.encode(message.destination, _source, _nonce, message.sender, message.target, message.message)
            );
            if (successfulMessages[messageHash]) {
                revert MessageAlreadyRelayed();
            }

            _storeMessageMetadata(_source, message.sender);

            bool success = SafeCall.call(message.target, 0, message.message);

            if (success) {
                successfulMessages[messageHash] = true;
                emit RelayedMessage(messageHash);
            } else {
                emit FailedRelayedMessage(messageHash);
            }

            _storeMessageMetadata(0, address(0));
        }
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

    /// @notice Internal function to append a new BatchableMessage to transient storage.
    /// @param _batchMessage The BatchableMessage struct to store.
    function _appendBatchMessage(BatchableMessage memory _batchMessage) internal {
        uint256 length;
        assembly {
            length := tload(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT)
        }

        bytes32 baseSlot = keccak256(abi.encodePacked(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT, length));
        address sender = _batchMessage.sender;
        address target = _batchMessage.target;
        uint256 destination = _batchMessage.destination;
        bytes memory message = _batchMessage.message;
        uint256 messageLength = _batchMessage.message.length;

        assembly {
            tstore(baseSlot, sender)
            tstore(add(baseSlot, 1), target)
            tstore(add(baseSlot, 2), destination)
            tstore(add(baseSlot, 3), messageLength)
        }

        for (uint256 i = 0; i < messageLength; i += 32) {
            bytes32 dataSlot = keccak256(abi.encodePacked(baseSlot, i / 32 + 4));
            bytes32 chunk;
            assembly {
                chunk := mload(add(message, add(0x20, i)))
                tstore(dataSlot, chunk)
            }
        }

        assembly {
            tstore(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT, add(length, 1))
        }
    }

    /// @notice Get the number of stored BatchableMessages in transient storage.
    /// @return The number of messages stored.
    function _getMessageCount() internal view returns (uint256) {
        uint256 length;
        assembly {
            length := tload(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT)
        }
        return length;
    }

    /// @notice Load a BatchableMessage from transient storage by index.
    /// @param index The index of the message to load.
    /// @return batchMessage The loaded BatchableMessage struct.
    function _loadMessage(uint256 index) internal view returns (BatchableMessage memory batchMessage) {
        uint256 length;

        // Load the current length of the array from transient storage
        assembly {
            length := tload(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT)
        }

        require(index < length, "Index out of bounds");

        // Calculate the base slot for the message at the given index
        bytes32 baseSlot = keccak256(abi.encodePacked(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT, index));

        address sender;
        address target;
        uint256 destination;
        uint256 messageLength;
        assembly {
            sender := tload(baseSlot)
            target := tload(add(baseSlot, 1))
            destination := tload(add(baseSlot, 2))
            messageLength := tload(add(baseSlot, 3))
        }

        batchMessage.sender = sender;
        batchMessage.target = target;
        batchMessage.destination = destination;
        bytes memory message = new bytes(messageLength);

        // Load the message bytes from transient storage
        for (uint256 i = 0; i < messageLength; i += 32) {
            bytes32 dataSlot = keccak256(abi.encodePacked(baseSlot, i / 32 + 4));
            bytes32 chunk;
            assembly {
                chunk := tload(dataSlot)
                mstore(add(message, add(0x20, i)), chunk)
            }
        }

        batchMessage.message = message;
    }

    /// @notice Helper function to iterate over all BatchableMessages stored in transient storage.
    function _processAllMessages() internal view returns (BatchableMessage[] memory) {
        uint256 messageCount = _getMessageCount();
        BatchableMessage[] memory allMessages = new BatchableMessage[](messageCount);

        for (uint256 i = 0; i < messageCount; i++) {
            BatchableMessage memory message = _loadMessage(i);
            allMessages[i] = message;
        }

        return allMessages;
    }

    /// @notice Clear a specific BatchableMessage stored in transient storage by index.
    /// @param index The index of the message to clear.
    function _clearMessage(uint256 index) internal {
        uint256 length;

        assembly {
            length := tload(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT)
        }

        require(index < length, "Index out of bounds");

        bytes32 baseSlot = keccak256(abi.encodePacked(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT, index));

        uint256 messageLength;
        assembly {
            messageLength := tload(add(baseSlot, 3)) // Load the message length
        }

        for (uint256 i = 0; i < messageLength; i += 32) {
            bytes32 dataSlot = keccak256(abi.encodePacked(baseSlot, i / 32 + 4));
            assembly {
                tstore(dataSlot, 0) // Clear the message data slots
            }
        }

        assembly {
            tstore(baseSlot, 0)
            tstore(add(baseSlot, 1), 0)
            tstore(add(baseSlot, 2), 0)
            tstore(add(baseSlot, 3), 0)
        }
    }

    /**
     * @notice Clear all BatchableMessages stored in transient storage.
     */
    function _clearAllMessages() internal {
        uint256 messageCount = _getMessageCount();

        for (uint256 i = 0; i < messageCount; i++) {
            _clearMessage(i);
        }

        assembly {
            tstore(CROSS_DOMAIN_BATCH_MESSAGE_COUNT_SLOT, 0)
        }
    }

    /// @notice Store batching status in transient storage.
    /// @param value The batching status.
    function _storeIsBatching(bool value) internal {
        // Use inline assembly to store the boolean value in transient storage
        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_IS_BATCHING_SLOT, value)
        }
    }
}
