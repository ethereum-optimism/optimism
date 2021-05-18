// SPDX-License-Identifier: MIT
pragma solidity >0.7.0 <0.9.0;
pragma experimental ABIEncoderV2;

contract MockL2CrossDomainMessenger {
    struct MessageData {
        address target;
        address sender;
        bytes message;
        uint256 messageNonce;
    }

    event SentMessage(bytes message);

    function emitSentMessageEvent(
        MessageData memory _message
    )
        public
    {
        emit SentMessage(
            abi.encodeWithSignature(
                "relayMessage(address,address,bytes,uint256)",
                _message.target,
                _message.sender,
                _message.message,
                _message.messageNonce
            )
        );
    }

    function emitMultipleSentMessageEvents(
        MessageData[] memory _messages
    )
        public
    {
        for (uint256 i = 0; i < _messages.length; i++) {
            emitSentMessageEvent(
                _messages[i]
            );
        }
    }
}
