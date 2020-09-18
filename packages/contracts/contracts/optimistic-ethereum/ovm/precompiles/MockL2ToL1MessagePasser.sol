pragma solidity ^0.5.0;

/* Interface Imports */
import { IL2ToL1MessagePasser } from "./L2ToL1MessagePasser.interface.sol";

/**
 * @title MockL2ToL1MessagePasser
 */
contract MockL2ToL1MessagePasser is IL2ToL1MessagePasser {
    /*
     * Contract Variables
     */

    mapping (bytes32 => bool) public storedMessages;

    /*
     * Public Functions
     */

    /**
     * Passes a message to L1.
     * @param _messageData Message to pass to L1.
     */
    function passMessageToL1(
        bytes memory _messageData
    )
        public
    {
        storedMessages[keccak256(_messageData)] = true;
    }
}