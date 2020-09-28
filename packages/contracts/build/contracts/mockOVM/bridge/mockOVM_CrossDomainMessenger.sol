// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { OVM_BaseCrossDomainMessenger } from "../../OVM/bridge/OVM_BaseCrossDomainMessenger.sol";

/**
 * @title mockOVM_CrossDomainMessenger
 */
contract mockOVM_CrossDomainMessenger is OVM_BaseCrossDomainMessenger {

    /***********
     * Structs *
     ***********/

    struct ReceivedMessage {
        uint256 timestamp;
        address target;
        address sender;
        bytes message;
        uint256 messageNonce;
        uint256 gasLimit;
    }


    /**********************
     * Contract Variables *
     **********************/

    ReceivedMessage[] internal fullReceivedMessages;
    uint256 internal lastRelayedMessage;
    uint256 internal delay;


    /***************
     * Constructor *
     ***************/

    /**
     * @param _delay Time in seconds before a message can be relayed.
     */
    constructor(
        uint256 _delay
    ) {
        delay = _delay;
    }


    /********************
     * Public Functions *
     ********************/

    /**
     * Sends a message to another mock xdomain messenger.
     * @param _target Target for the message.
     * @param _message Message to send.
     * @param _gasLimit Amount of gas to send with the call.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    )
        public
    {
        mockOVM_CrossDomainMessenger targetMessenger = mockOVM_CrossDomainMessenger(
            targetMessengerAddress
        );

        // Just send it over!
        targetMessenger.receiveMessage(ReceivedMessage({
            timestamp: block.timestamp,
            target: _target,
            sender: msg.sender,
            message: _message,
            messageNonce: messageNonce,
            gasLimit: _gasLimit
        }));

        messageNonce += 1;
    }

    /**
     * Receives a message to be sent later.
     * @param _message Message to send later.
     */
    function receiveMessage(
        ReceivedMessage memory _message
    )
        public
    {
        fullReceivedMessages.push(_message);
    }

    /**
     * Checks whether we have messages to relay.
     * @param _exists Whether or not we have more messages to relay.
     */
    function hasNextMessage()
        public
        view
        returns (
            bool _exists
        )
    {
        return fullReceivedMessages.length < lastRelayedMessage;
    }

    /**
     * Relays the last received message not yet relayed.
     */
    function relayNextMessage()
        public
    {
        if (hasNextMessage() == false) {
            return;
        }

        ReceivedMessage memory nextMessage = fullReceivedMessages[lastRelayedMessage];
        if (nextMessage.timestamp + delay > block.timestamp) {
            return;
        }

        xDomainMessageSender = nextMessage.sender;
        nextMessage.target.call{gas: nextMessage.gasLimit}(nextMessage.message);

        lastRelayedMessage += 1;
    }
}
