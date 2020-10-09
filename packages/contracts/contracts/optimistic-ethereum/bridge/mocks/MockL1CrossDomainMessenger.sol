// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { MockCrossDomainMessenger } from "./MockCrossDomainMessenger.sol";

/**
 * @title MockL1CrossDomainMessenger
 */
contract MockL1CrossDomainMessenger is MockCrossDomainMessenger {

    event RelayedL2ToL1Message(bytes32 messageHash);



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
        MockCrossDomainMessenger(_delay)
    {}

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
        bytes32 messageHash = getReceivedMessageHash(nextMessage);
        emit RelayedL2ToL1Message(messageHash);
    }
}
