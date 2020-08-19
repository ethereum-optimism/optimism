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
     * Public Functions
     */

    function passMessageToL1(
        bytes memory _messageData,
        address l1TargetAddress
    ) public {

    }
}