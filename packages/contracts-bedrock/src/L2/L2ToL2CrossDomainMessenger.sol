// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import { SafeCall } from "src/libraries/SafeCall.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @custom:upgradeable
/// @title L2ToL2CrossDomainMessenger
contract L2ToL2CrossDomainMessenger {
    /// @notice Slot for the sender of the current cross domain message.
    /// @dev Equal to bytes32(uint256(keccak256("l2tol2crossdomainmessenger.sender")) - 1)
    bytes32 public constant CROSS_DOMAIN_MESSAGE_SENDER_SLOT =
        0xb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f3;

    /// @notice Current message version identifier.
    uint16 public constant MESSAGE_VERSION = uint16(0);

    /// @notice Initial balance for the contract.
    uint248 public constant INITIAL_BALANCE = type(uint248).max;

    /// @notice Address of the L2 Cross Domain Messenger on this chain.
    address public immutable CROSS_L2_INBOX;

    /// @notice Mapping of message hashes to boolean receipt values. Note that a message will only
    ///         be present in this mapping if it has successfully been relayed on this chain, and
    ///         can therefore not be relayed again.
    mapping(bytes32 => bool) public successfulMessages;

    /// @notice Nonce for the next message to be sent, without the message version applied. Use the
    ///         messageNonce getter which will insert the message version into the nonce to give you
    ///         the actual nonce to be used for the message.
    uint240 internal msgNonce;

    /// @notice Emitted whenever a message is sent to the other chain.
    /// @param destination Chain ID of the destination chain.
    /// @param target      Target contract or wallet address.
    /// @param message     Message to trigger the target address with.
    /// @param data        Data to be sent with the message.
    event SentMessage(uint256 destination, address target, bytes message, bytes data) anonymous;

    constructor(address _crossL2Inbox) {
        CROSS_L2_INBOX = _crossL2Inbox;
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
        require(_destination != block.chainid);

        bytes memory data = abi.encodeCall(
            L2ToL2CrossDomainMessenger.relayMessage,
            (_destination, messageNonce(), msg.sender, _target, msg.value, _message)
        );
        emit SentMessage(_destination, _target, _message, data);
        msgNonce++;
    }

    /// @notice Relays a message that was sent by the other CrossDomainMessenger contract. Can only
    ///         be executed via cross-chain call from the other messenger OR if the message was
    ///         already received once and is currently being replayed.
    /// @param _destination Chain ID of the destination chain.
    /// @param _nonce       Nonce of the message being relayed.
    /// @param _sender      Address of the user who sent the message.
    /// @param _target      Address that the message is targeted at.
    /// @param _value       ETH value to send with the message.
    /// @param _message     Message to send to the target.
    function relayMessage(
        uint256 _destination,
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        bytes memory _message
    )
        external
    {
        require(msg.sender == CROSS_L2_INBOX);
        require(CrossL2Inbox(CROSS_L2_INBOX).origin() == address(this));
        require(_destination == block.chainid);
        require(_target != address(this));

        bytes32 messageHash = keccak256(abi.encode(_destination, _nonce, _sender, _target, _value, _message));
        require(successfulMessages[messageHash] == false);

        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)
        }

        bool success = SafeCall.call({ _target: _target, _gas: gasleft(), _value: _value, _calldata: _message });

        require(success);

        successfulMessages[messageHash] = true;
    }
}
