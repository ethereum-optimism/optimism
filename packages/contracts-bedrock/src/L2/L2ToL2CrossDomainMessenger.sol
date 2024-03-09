// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { ISemver } from "src/universal/ISemver.sol";

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000023
/// @title L2ToL2CrossDomainMessenger
/// @notice The L2ToL2CrossDomainMessenger is a higher level abstraction on top of the CrossL2Inbox that provides
///         features necessary for secure transfers ERC20 tokens between L2 chains. Messages sent through the
///         L2ToL2CrossDomainMessenger on the source chain receive both replay protection as well as domain binding.
contract L2ToL2CrossDomainMessenger is ISemver {
    /// @notice Storage slot for the sender of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.sender")) - 1)
    bytes32 public constant CROSS_DOMAIN_MESSAGE_SENDER_SLOT =
        0xb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f3;

    /// @notice Storage slot for the source of the current cross domain message.
    ///         Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.source")) - 1)
    bytes32 public constant CROSS_DOMAIN_MESSAGE_SOURCE_SLOT =
        0x711dfa3259c842fffc17d6e1f1e0fc5927756133a2345ca56b4cb8178589fee7;

    /// @notice Current message version identifier.
    uint16 public constant MESSAGE_VERSION = uint16(0);

    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only
    ///         be present in this mapping if it has successfully been relayed on this chain, and
    ///         can therefore not be relayed again.
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Nonce for the next message to be sent, without the message version applied. Use the
    ///         messageNonce getter which will insert the message version into the nonce to give you
    ///         the actual nonce to be used for the message.
    uint240 internal msgNonce;

    /// @notice Emitted whenever a message is sent to the other chain.
    /// @param data Encoded data of the message that was sent.
    event SentMessage(bytes data) anonymous;

    /// @notice Emitted whenever a message is successfully relayed on this chain.
    /// @param msgHash Hash of the message that was relayed.
    event RelayedMessage(bytes32 indexed msgHash);

    /// @notice Emitted whenever a message fails to be relayed on this chain.
    /// @param msgHash Hash of the message that failed to be relayed.
    event FailedRelayedMessage(bytes32 indexed msgHash);

    /// @notice Retrieves the sender of the current cross domain message.
    /// @return _sender Address of the sender of the current cross domain message.
    function crossDomainMessageSender() external view returns (address _sender) {
        assembly {
            _sender := tload(CROSS_DOMAIN_MESSAGE_SENDER_SLOT)
        }
    }

    /// @notice Retrieves the source of the current cross domain message.
    /// @return _source Chain ID of the source of the current cross domain message.
    function crossDomainMessageSource() external view returns (uint256 _source) {
        assembly {
            _source := tload(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT)
        }
    }

    /// @notice Retrieves the next message nonce. Message version will be added to the upper two
    ///         bytes of the message nonce. Message version allows us to treat messages as having
    ///         different structures.
    /// @return Nonce of the next message to be sent, with added message version.
    function messageNonce() public view returns (uint256) {
        return Encoding.encodeVersionedNonce(msgNonce, MESSAGE_VERSION);
    }

    /// @notice Sends a message to some target address on a destination chain. Note that if the call
    ///         always reverts, then the message will be unrelayable, and any ETH sent will be
    ///         permanently locked. The same will occur if the target on the other chain is
    ///         considered unsafe (see the _isUnsafeTarget() function).
    /// @param _destination Chain ID of the destination chain.
    /// @param _target      Target contract or wallet address.
    /// @param _message     Message to trigger the target address with.
    function sendMessage(uint256 _destination, address _target, bytes calldata _message) external payable {
        require(_destination != block.chainid, "L2ToL2CrossDomainMessenger: cannot send message to self");

        bytes memory data = abi.encodeCall(
            L2ToL2CrossDomainMessenger.relayMessage,
            (_destination, block.chainid, messageNonce(), msg.sender, _target, _message)
        );
        emit SentMessage(data);
        msgNonce++;
    }

    /// @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only
    ///         be executed via cross-chain call from the other messenger OR if the message was
    ///         already received once and is currently being replayed.
    /// @param _destination Chain ID of the destination chain.
    /// @param _source      Chain ID of the source chain.
    /// @param _nonce       Nonce of the message being relayed.
    /// @param _sender      Address of the user who sent the message.
    /// @param _target      Address that the message is targeted at.
    /// @param _message     Message to send to the target.
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
    {
        require(msg.sender == Predeploys.CROSS_L2_INBOX, "L2ToL2CrossDomainMessenger: sender not CrossL2Inbox");
        require(
            CrossL2Inbox(Predeploys.CROSS_L2_INBOX).origin() == address(this),
            "L2ToL2CrossDomainMessenger: CrossL2Inbox origin not this contract"
        );
        require(_destination == block.chainid, "L2ToL2CrossDomainMessenger: destination not this chain");
        require(_target != Predeploys.CROSS_L2_INBOX, "L2ToL2CrossDomainMessenger: CrossL2Inbox cannot call itself");

        bytes32 messageHash = keccak256(abi.encode(_destination, _source, _nonce, _sender, _target, _message));
        require(successfulMessages[messageHash] == false, "L2ToL2CrossDomainMessenger: message already relayed");

        bool success;
        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SOURCE_SLOT, _source)
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)

            success :=
                call(
                    gas(), // gas
                    _target, // recipient
                    callvalue(), // ether value
                    add(_message, 32), // inloc
                    mload(_message), // inlen
                    0, // outloc
                    0 // outlen
                )
        }

        if (success) {
            successfulMessages[messageHash] = true;
            emit RelayedMessage(messageHash);
        } else {
            emit FailedRelayedMessage(messageHash);
        }
    }
}
