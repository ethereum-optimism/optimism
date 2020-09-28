// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { BaseCrossDomainMessenger } from "../BaseCrossDomainMessenger.sol";

/**
 * @title MockCrossDomainMessenger
 */
contract MockCrossDomainMessenger is BaseCrossDomainMessenger {

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
    )
        public
    {
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
        MockCrossDomainMessenger targetMessenger = MockCrossDomainMessenger(
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
        if (fullReceivedMessages.length <= lastRelayedMessage) {
            return false;
        }

        ReceivedMessage memory nextMessage = fullReceivedMessages[lastRelayedMessage];
        return nextMessage.timestamp + delay < block.timestamp;
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
        xDomainMessageSender = nextMessage.sender;
        nextMessage.target.call.gas(nextMessage.gasLimit)(nextMessage.message);
        lastRelayedMessage += 1;
    }
}
