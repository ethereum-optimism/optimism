pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract IL2ToL1MessagePasser {
    /*
     * Events
     */

    event L2ToL1Message(
       uint _nonce,
       address _ovmSender,
       bytes _callData
    );


    /*
     * Contract Variables
     */

    uint nonce;
    address executionManagerAddress;


    /*
     * Constructor
     */

    // constructor(
    //     address _executionManagerAddress
    // ) public {
    //     executionManagerAddress = _executionManagerAddress;
    // }

    function passMessageToL1(
        bytes memory _messageData
    ) public;
}