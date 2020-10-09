// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { ICrossDomainMessenger } from "../interfaces/CrossDomainMessenger.interface.sol";

/**
 * @title MockCrossDomainMessenger
 */
contract MockCrossDomainMessenger is ICrossDomainMessenger {

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

    mapping (bytes32 => bool) public receivedMessages;
    mapping (bytes32 => bool) public sentMessages;
    address public targetMessengerAddress;
    uint256 public messageNonce;
    address public xDomainMessageSender;



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
     * Sets the target messenger address.
     * @param _targetMessengerAddress New messenger address.
     */
    function setTargetMessengerAddress(
        address _targetMessengerAddress
    )
        external
    {
        targetMessengerAddress = _targetMessengerAddress;
    }

    /**
     * Sends a message to another mock xdomain messenger.
     * @param _target Target for the message.
     * @param _message Message to send.
     * @param _gasLimit Amount of gas to send with the call.
     */
    function sendMessage(
        address _target,
        bytes calldata _message,
        uint32 _gasLimit
    )
        external
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
        return true; //nextMessage.timestamp + delay < block.timestamp;
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
