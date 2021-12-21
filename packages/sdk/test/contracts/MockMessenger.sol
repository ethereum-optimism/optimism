pragma solidity ^0.8.9;

import { ICrossDomainMessenger } from "./ICrossDomainMessenger.sol";

contract MockMessenger is ICrossDomainMessenger {
    function xDomainMessageSender() public view returns (address) {
        return address(0);
    }

    // Empty function to satisfy the interface.
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _gasLimit
    ) public {
        return;
    }

    struct SentMessageEventParams {
        address target;
        address sender;
        bytes message;
        uint256 messageNonce;
        uint256 gasLimit;
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
            _params.gasLimit
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
