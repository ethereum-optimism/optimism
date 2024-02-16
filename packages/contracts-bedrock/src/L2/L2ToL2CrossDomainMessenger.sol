// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";
import { CrossL2Inbox } from "src/L2/CrossL2Inbox.sol";
import { Encoding } from "src/libraries/Encoding.sol";

/// @custom:upgradeable
/// @title L2ToL2CrossDomainMessenger
abstract contract L2ToL2CrossDomainMessenger is Initializable, CrossL2Inbox {
    uint16 public constant MESSAGE_VERSION = 1;
    uint240 public nonce;

    mapping(bytes32 => bool) public sentMessages;

    // bytes32(uint256(keccak256("l2tol2crossdomainmessenger.sender")) - 1)
    bytes32 internal constant CROSS_DOMAIN_MESSAGE_SENDER_SLOT =
        0xb83444d07072b122e2e72a669ce32857d892345c19856f4e7142d06a167ab3f3;

    CrossL2Inbox public constant CROSS_L2_INBOX = CrossL2Inbox(0x4200000000000000000000000000000000000022);

    event SentMessage(bytes message) anonymous;

    function messageNonce() public view returns (uint256) {
        return Encoding.encodeVersionedNonce(nonce, MESSAGE_VERSION);
    }

    function sendMessage(uint256 _destination, address _target, bytes calldata _message) external payable {
        require(_destination != block.chainid);

        bytes memory data = abi.encodeCall(
            L2ToL2CrossDomainMessenger.relayMessage,
            (_destination, messageNonce(), msg.sender, _target, msg.value, _message)
        );
        emit SentMessage(data);
        nonce++;
    }

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
        require(msg.sender == address(CROSS_L2_INBOX));
        require(_destination == block.chainid);
        require(CROSS_L2_INBOX.origin() == address(this));
        require(_target != address(this));

        bytes32 messageHash = keccak256(abi.encode(_destination, _nonce, _sender, _target, _value, _message));
        require(sentMessages[messageHash] == false);

        assembly {
            tstore(CROSS_DOMAIN_MESSAGE_SENDER_SLOT, _sender)
        }

        bool success = SafeCall.call({ _target: _target, _gas: gasleft(), _value: _value, _calldata: _message });

        require(success);

        sentMessages[messageHash] = true;
    }
}
