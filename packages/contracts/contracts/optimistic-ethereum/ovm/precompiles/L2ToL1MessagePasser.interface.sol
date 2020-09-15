pragma solidity ^0.5.0;

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
     * Public Functions
     */
    
    function passMessageToL1(bytes memory _messageData) public;
}