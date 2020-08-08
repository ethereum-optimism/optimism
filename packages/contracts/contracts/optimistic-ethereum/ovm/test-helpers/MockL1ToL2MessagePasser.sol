pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

//import { IL2ToL1MessagePasser } from "../../bridge/IL2ToL1MessagePasser.sol";

contract MockL1ToL2MessagePasser {
    /*
     * Events
     */

    event L1ToL2Message(
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

    // constructor(address _executionManagerAddress) public {
    //     executionManagerAddress = _executionManagerAddress;
    // }


    /*
     * Public Functions
     */

    function passMessageToL2(bytes memory _messageData) public {
        // For now, to be trustfully relayed

        emit L1ToL2Message(
            nonce++,
            msg.sender,
            _messageData
        );
    }
}