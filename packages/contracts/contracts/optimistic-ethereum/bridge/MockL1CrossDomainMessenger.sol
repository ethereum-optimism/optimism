pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Contract Imports */
import { L1CrossDomainMessenger } from "./L1CrossDomainMessenger.sol";

/**
 * @title MockL1CrossDomainMessenger
 */
contract MockL1CrossDomainMessenger is L1CrossDomainMessenger {
    /*
     * Internal Functions
     */

    /**
     * Verifies that the given message is valid.
     * .inheritdoc L1CrossDomainMessenger
     */
    function _verifyXDomainMessage(
        bytes memory _xDomainCalldata,
        L2MessageInclusionProof memory _proof
    )
        internal
        returns (
            bool
        )
    {
        return true;
    }

    /**
     * Sends a cross domain message.
     * .inheritdoc L1CrossDomainMessenger
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint32 _gasLimit
    )
        internal
    {
        return;
    }
}