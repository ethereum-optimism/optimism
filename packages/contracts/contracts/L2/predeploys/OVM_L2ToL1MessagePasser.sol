// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

/* Interface Imports */
import { iOVM_L2ToL1MessagePasser } from "./iOVM_L2ToL1MessagePasser.sol";

/**
 * @title OVM_L2ToL1MessagePasser
 * @dev The L2 to L1 Message Passer is a utility contract which facilitate an L1 proof of the
 * of a message on L2. The L1 Cross Domain Messenger performs this proof in its
 * _verifyStorageProof function, which verifies the existence of the transaction hash in this
 * contract's `sentMessages` mapping.
 */
contract OVM_L2ToL1MessagePasser is iOVM_L2ToL1MessagePasser {
    /**********************
     * Contract Variables *
     **********************/

    mapping(bytes32 => bool) public sentMessages;

    /********************
     * Public Functions *
     ********************/

    /**
     * Passes a message to L1.
     * @param _message Message to pass to L1.
     */
    // slither-disable-next-line external-function
    function passMessageToL1(bytes memory _message) public {
        // Note: although this function is public, only messages sent from the
        // L2CrossDomainMessenger will be relayed by the L1CrossDomainMessenger.
        // This is enforced by a check in L1CrossDomainMessenger._verifyStorageProof().
        sentMessages[keccak256(abi.encodePacked(_message, msg.sender))] = true;
    }
}
