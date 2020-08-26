pragma solidity ^0.5.0;

/* Contract Imports */
import { BaseMessenger } from "./BaseMessenger.sol";
import { L1MessageSender } from "../ovm/precompiles/L1MessageSender.sol";
import { L2ToL1MessagePasser } from "../ovm/precompiles/L2ToL1MessagePasser.sol";

contract L2ToL1Messenger is BaseMessenger {
    /*
     * Internal Functions
     */

    /**
     * Verifies that a received cross domain message is valid.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return whether or not the message is valid.
     */
    function _verifyXDomainMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
        returns (
            bool
        )
    {
        L1MessageSender l1MessageSenderPrecompile = L1MessageSender(0x4200000000000000000000000000000000000001);
        address l1MessageSenderAddress = l1MessageSenderPrecompile.getL1MessageSender();
        return l1MessageSenderAddress == targetMessengerAddress;
    }

    /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit Gas limit for the message. Unused when sending from L2 to L1.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint32 _gasLimit
    )
        internal
    {
        L2ToL1MessagePasser l2ToL1MessagePasserPrecompile = L2ToL1MessagePasser(0x4200000000000000000000000000000000000000);
        l2ToL1MessagePasserPrecompile.passMessageToL1(_message);
    }
}
