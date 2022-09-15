pragma solidity ^0.8.9;

import { ICrossDomainMessenger } from "./ICrossDomainMessenger.sol";

contract MockMessenger is ICrossDomainMessenger {
    function xDomainMessageSender() public view returns (address) {
        return address(0);
    }

    uint256 public nonce;

    // Empty function to satisfy the interface.
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _gasLimit
    ) public {
        emit SentMessage(
            _target,
            msg.sender,
            _message,
            nonce,
            _gasLimit
        );
        nonce++;
    }

    function replayMessage(
        address _target,
        address _sender,
        bytes calldata _message,
        uint256 _queueIndex,
        uint32 _oldGasLimit,
        uint32 _newGasLimit
    ) public {
        emit SentMessage(
            _target,
            _sender,
            _message,
            nonce,
            _newGasLimit
        );
        nonce++;
    }

    struct SentMessageEventParams {
        address target;
        address sender;
        bytes message;
        uint256 messageNonce;
        uint256 minGasLimit;
        uint256 value;
    }

    function doNothing() public {
        return;
    }

    function triggerSentMessageEvent(
        SentMessageEventParams memory _params
    ) public {
        emit SentMessage(
            _params.target,
            _params.sender,
            _params.message,
            _params.messageNonce,
            _params.minGasLimit
        );
    }

    function triggerSentMessageEvents(
        SentMessageEventParams[] memory _params
    ) public {
        for (uint256 i = 0; i < _params.length; i++) {
            triggerSentMessageEvent(_params[i]);
        }
    }

    function triggerRelayedMessageEvents(
        bytes32[] memory _params
    ) public {
        for (uint256 i = 0; i < _params.length; i++) {
            emit RelayedMessage(_params[i]);
        }
    }

    function triggerFailedRelayedMessageEvents(
        bytes32[] memory _params
    ) public {
        for (uint256 i = 0; i < _params.length; i++) {
            emit FailedRelayedMessage(_params[i]);
        }
    }
}
