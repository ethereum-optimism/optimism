// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { iOVM_BaseCrossDomainMessenger } from "../../iOVM/bridge/iOVM_BaseCrossDomainMessenger.sol";

/**
 * @title OVM_BaseCrossDomainMessenger
 */
contract OVM_BaseCrossDomainMessenger is iOVM_BaseCrossDomainMessenger {

    /**********************
     * Contract Variables *
     **********************/

    mapping (bytes32 => bool) public receivedMessages;
    mapping (bytes32 => bool) public sentMessages;
    address public targetMessengerAddress;
    uint256 public messageNonce;
    address public xDomainMessageSender;


    /********************
     * Public Functions *
     ********************/

    /**
     * Sets the target messenger address.
     * @dev Currently, this function is public and therefore allows anyone to modify the target
     *      messenger for a given xdomain messenger contract. Obviously this shouldn't be allowed,
     *      but we still need to determine an adequate mechanism for updating this address.
     * @param _targetMessengerAddress New messenger address.
     */
    function setTargetMessengerAddress(
        address _targetMessengerAddress
    )
        override
        public
    {
        targetMessengerAddress = _targetMessengerAddress;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint256 _gasLimit
    )
        override
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            msg.sender,
            _message,
            messageNonce
        );

        _sendXDomainMessage(xDomainCalldata, _gasLimit);

        messageNonce += 1;
        sentMessages[keccak256(xDomainCalldata)] = true;
    }

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Generates the correct cross domain calldata for a message.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return ABI encoded cross domain calldata.
     */
    function _getXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
        pure
        returns (
            bytes memory
        )
    {
        return abi.encodeWithSelector(
            bytes4(keccak256(bytes("relayMessage(address,address,bytes,uint256)"))),
            _target,
            _sender,
            _message,
            _messageNonce
        );
    }

    /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit Gas limit for the provided message.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint256 _gasLimit
    )
        virtual
        internal
    {
        revert("Implement me in child contracts!");
    }
}
