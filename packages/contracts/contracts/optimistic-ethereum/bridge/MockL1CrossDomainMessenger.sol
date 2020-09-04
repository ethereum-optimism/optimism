pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Interface Imports */
import { IL2CrossDomainMessenger } from "./L2CrossDomainMessenger.interface.sol";

/* Contract Imports */
import { BaseMockCrossDomainMessenger } from "./BaseMockCrossDomainMessenger.sol";
import { L1CrossDomainMessenger } from "./L1CrossDomainMessenger.sol";

/**
 * @title MockL1CrossDomainMessenger
 */
contract MockL1CrossDomainMessenger is BaseMockCrossDomainMessenger, L1CrossDomainMessenger {
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
     * Internal relay function.
     */
    function _relayXDomainMessageToTarget(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
    {
        IL2CrossDomainMessenger(targetMessengerAddress).relayMessage(
            _target,
            _sender,
            _message,
            _messageNonce
        );
    }
}
