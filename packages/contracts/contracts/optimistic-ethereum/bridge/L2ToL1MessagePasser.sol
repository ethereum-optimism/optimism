pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;
import { IL2ToL1MessagePasser } from "../bridge/IL2ToL1MessagePasser.sol";

contract L2ToL1MessagePasser is IL2ToL1MessagePasser{

    mapping (uint => bytes32) public storedMessages;
    uint public index = 0;
    uint public lastTimestampUsed = 0;

    function passMessageToL1(
        bytes memory _messageData,
        address l1TargetAddress
    ) public {
        // To avoid bloat, overwrite messages every block
        if (lastTimestampUsed != block.timestamp) {
            index = 0;
            lastTimestampUsed = block.timestamp;
        }
        storedMessages[index] = keccak256(
            abi.encode(
                msg.sender,
                _messageData,
                l1TargetAddress
            )
        );
        index ++;
    }
}