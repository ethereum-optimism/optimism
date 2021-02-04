// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/* Interface Imports */
import { iOVM_L2ToL1MessagePasser } from "../../iOVM/precompiles/iOVM_L2ToL1MessagePasser.sol";

/**
 * @title OVM_L2ToL1MessagePasser
 * @dev The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the 
 * of a message on L2. The L1 Cross Domain Messenger performs this proof in its
 * _verifyStorageProof function, which verifies the existence of the transaction hash in this 
 * contract's `sentMessages` mapping.
 * 
 * Compiler used: solc
 * Runtime target: EVM
 */
contract OVM_L2ToL1MessagePasser is iOVM_L2ToL1MessagePasser {

    /**********************
     * Contract Variables *
     **********************/

    mapping (bytes32 => bool) public sentMessages;


    /********************
     * Public Functions *
     ********************/

    /**
     * Passes a message to L1.
     * @param _message Message to pass to L1.
     */
    function passMessageToL1(
        bytes memory _message
    )
        override
        public
    {
        // Note: although this function is public, only messages sent from the OVM_L2CrossDomainMessenger 
        // will be relayed by the OVM_L1CrossDomainMessenger. This is enforced by a check in 
        // OVM_L1CrossDomainMessenger._verifyStorageProof().
        sentMessages[keccak256(
            abi.encodePacked(
                _message,
                msg.sender
            )
        )] = true;
    }
}
