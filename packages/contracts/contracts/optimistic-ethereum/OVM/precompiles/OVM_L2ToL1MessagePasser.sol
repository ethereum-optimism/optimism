// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.7.0;

/* Interface Imports */
import { iOVM_L2ToL1MessagePasser } from "../../iOVM/precompiles/iOVM_L2ToL1MessagePasser.sol";
import { iOVM_ExecutionManager } from "../../iOVM/execution/iOVM_ExecutionManager.sol";

/**
 * @title OVM_L2ToL1MessagePasser
 */
contract OVM_L2ToL1MessagePasser is iOVM_L2ToL1MessagePasser {

    /**********************
     * Contract Variables *
     **********************/

    uint256 internal nonce;


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
        // For now, to be trustfully relayed by sequencer to L1, so just emit
        // an event for the sequencer to pick up.
        emit L2ToL1Message(
            nonce,
            iOVM_ExecutionManager(msg.sender).ovmCALLER(),
            _message
        );

        nonce = nonce + 1;
    }
}
