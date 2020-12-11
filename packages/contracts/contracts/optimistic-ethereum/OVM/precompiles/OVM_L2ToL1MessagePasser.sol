// SPDX-License-Identifier: MIT
// +build ovm
pragma solidity >0.6.0 <0.8.0;

/* Interface Imports */
import { iOVM_L2ToL1MessagePasser } from "../../iOVM/precompiles/iOVM_L2ToL1MessagePasser.sol";

/**
 * @title OVM_L2ToL1MessagePasser
 * @dev L2 CONTRACT (COMPILED)
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
        sentMessages[keccak256(
            abi.encodePacked(
                _message,
                msg.sender
            )
        )] = true;
    }
}
