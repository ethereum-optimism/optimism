pragma solidity ^0.5.0;
pragma experimental ABIEncoderV2;

/* Internal Imports */
import {DataTypes as dt} from "./DataTypes.sol";

contract L1ToL2MessagePasser {
    uint nonce = 0;

    event L1ToL2Message(
        uint _nonce,
        address _sender,
        address _target,
        bytes _callData
    );

    function passMessageToL2(address ovmEntrypoint, bytes memory ovmCalldata) public {
        // TODO: Actually create/enqueue a rollup block with this message.  We are simply mocking this functionality for now.
        emit L1ToL2Message(
            nonce++,
            msg.sender,
            ovmEntrypoint,
            ovmCalldata
        );
    }
}