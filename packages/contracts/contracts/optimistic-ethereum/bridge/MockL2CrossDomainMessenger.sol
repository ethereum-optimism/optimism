pragma solidity ^0.5.0;

/* Contract Imports */
import { L2CrossDomainMessenger } from "./L2CrossDomainMessenger.sol";

/**
 * @title MockL2CrossDomainMessenger
 */
contract MockL2CrossDomainMessenger is L2CrossDomainMessenger {
    /*
     * Internal Functions
     */

    /**
     * Verifies that a received cross domain message is valid.
     * .inheritdoc L2CrossDomainMessenger
     */
    function _verifyXDomainMessage()
        internal
        returns (
            bool
        )
    {
        return true;
    }

    /**
     * Sends a cross domain message.
     * .inheritdoc L2CrossDomainMessenger
     */
    function _sendXDomainMessage(
        bytes memory _message
    )
        internal
    {
        return;
    }
}