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

    function triggerSentMessageEvents(
        SentMessageEventParams[] memory _params
    ) public {
        for (uint256 i = 0; i < _params.length; i++) {
            emit SentMessage(
                _params[i].target,
                _params[i].sender,
                _params[i].message,
                _params[i].messageNonce,
                _params[i].gasLimit
            );
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
