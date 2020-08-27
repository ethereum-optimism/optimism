pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { IL1MessageSender } from "./L1MessageSender.interface.sol";

/**
 * @title MockL1MessageSender
 */
contract MockL1MessageSender is IL1MessageSender {
    /*
     * Contract Variables
     */

    address private l1MessageSender;


    /*
     * Public Functions
     */

    /**
     * @return L1 message sender address (msg.sender).
     */
    function getL1MessageSender()
        public
        returns (address)
    {
        return l1MessageSender;
    }

    /**
     * Sets the L1 message sender address.
     * @param _l1MessageSender L1 message sender address.
     */
    function setL1MessageSender(
        address _l1MessageSender
    )
        public
    {
        l1MessageSender = _l1MessageSender;
    }
}