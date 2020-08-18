pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

contract IL2ToL1MessagePasser {

    mapping (uint => bytes32) public storedMessages;
    uint public index;
    uint public lastTimestampUsed;

    function passMessageToL1(
        bytes memory _messageData,
        address l1TargetAddress
    ) public;
}