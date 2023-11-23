// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";

import { Constants } from "src/libraries/Constants.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

contract InteropEnabledL2CrossDomainMessenger {
    // new message versionn
    uint16 public constant INTEROP_MESSAGE_VERSION = 2;

    bytes32 public constant ETH_MAINNET_ID = bytes32(hex"01");
    bytes32 public CHAIN_ID;

    constructor(bytes32 _chain_id) {
        CHAIN_ID = _chain_id;
    }

    // TODO: Events

    /**
     * CrossDomainMessenger: some copied logic & updated/extended dispatch interface
     */

    uint240 internal msgNonce;
    address internal xDomainMsgSender;
    mapping(bytes32 => bool) public successfulMessages;
    mapping(bytes32 => bool) public failedMessages;

    function messageNonce() public view returns (uint256) {
        return Encoding.encodeVersionedNonce(msgNonce, INTEROP_MESSAGE_VERSION);
    }

    function sendMessage(bytes32 _destination, address _target, bytes calldata _message, uint32 _minGasLimit) external payable {
        if (_destination == ETH_MAINNET_ID) {
            // Required as we are leaving the CrossDomainMessenger untouched. L2->L2 messages will still be using
            // the old message format (Version 1).
            L2CrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).sendMessage(_target, _message, _minGasLimit);
            return;
        }

        require(_destination != CHAIN_ID, "message cant be sent to self");

        bytes memory data = abi.encodeWithSelector(
            this.relayMessage.selector, messageNonce(), CHAIN_ID, msg.sender, _target, msg.value, _minGasLimit, _message
        );

        // (1) Send to the CrossL2Inbox.
        // ***Note*** With an updated CrossDomainMessenger message format, this contract can replace the L2CDM predeploy
        // and switch between L2ToL1MessagePasser/CrossL2Inbox based on `_destination`.
        //
        // i.e: CrossL2Inbox.sendMessage(source, destination, _minGasLimit, msg.value, data) // ensure _minGaslimit covers the base

        // (2) Emit Events

        unchecked {
            ++msgNonce;
        }
    }

    function relayMessage(
        uint256 _nonce,
        bytes32 _source,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes calldata _message
    ) external payable {
        (, uint16 version) = Encoding.decodeVersionedNonce(_nonce);
        require(version == INTEROP_MESSAGE_VERSION, "message is not of the right version");

        bytes32 messageHash = hashCrossDomainMessageV2(_nonce, _source, CHAIN_ID, _sender, _target, _value, _minGasLimit, _message);
        require(successfulMessages[messageHash] == false, "message already processed");

        // (1) Allow for replay
        if (_sender == address(this)) {
            assert(msg.value == _value);
            assert(failedMessages[messageHash] == false);
        } else {
            require(msg.value == 0, "cannot replay with more funds");
            require(failedMessages[messageHash], "message cannot be replayed");
        }

        // .. check for min gas, unsafe target, etc.

        // (2) Relay Message
        // -- Ignoring the reserved gas subtracted from gasleft() & xDomainMsgSender update
        xDomainMsgSender = _sender;
        bool success = SafeCall.call(_target, gasleft(), _value, _message);
        xDomainMsgSender = Constants.DEFAULT_L2_SENDER;
        if (success) {
            successfulMessages[messageHash] = true;
            // (3) Emit event
        } else {
            failedMessages[messageHash] = true;
            // (3) Emit event
        }
    }

    
    /**
     * Encoding.sol: update for this new message version
     */

    function encodeCrossDomainMessageV2(
        uint256 _nonce,
        bytes32 _source,
        bytes32 _destination,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes memory) {
        return abi.encodeWithSignature(
            "relayMessage(uint256,bytes32,bytes32,address,address,uint256,uint256,bytes)",
            _nonce,
            _source,
            _destination,
            _sender,
            _target,
            _value,
            _gasLimit,
            _data
        );
    }

    /**
     * Hashing.sol: update for this new message version
     */
    function hashCrossDomainMessageV2(
        uint256 _nonce,
        bytes32 _source,
        bytes32 _destination,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _gasLimit,
        bytes memory _data
    ) internal pure returns (bytes32) {
        return keccak256(encodeCrossDomainMessageV2(_nonce, _source, _destination, _sender, _target, _value, _gasLimit, _data));
    }
}
