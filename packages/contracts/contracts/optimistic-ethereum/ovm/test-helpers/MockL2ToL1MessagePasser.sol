pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

import { IL2ToL1MessagePasser } from "../../bridge/IL2ToL1MessagePasser.sol";

contract MockL2ToL1MessagePasser is IL2ToL1MessagePasser {


    /*
     * Contract Variables
     */

    uint nonce;
    address executionManagerAddress;


    /*
     * Constructor
     */

    // constructor(address _executionManagerAddress) public {
    //     executionManagerAddress = _executionManagerAddress;
    // }


    /*
     * Public Functions
     */

    function passMessageToL1(bytes memory _messageData) public {
        // For now, to be trustfully relayed by sequencer to L1, so just emit
        // an event for the sequencer to pick up.
    }
}