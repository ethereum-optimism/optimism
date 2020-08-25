pragma solidity ^0.5.0;

import { MockCrossDomainMessageReceiver } from "../optimistic-ethereum/bridge/MockCrossDomainMessageReceiver.sol";
import { SimpleStorage } from "./SimpleStorage.sol";

contract SimpleStorageMessageReceiver is SimpleStorage, MockCrossDomainMessageReceiver {
    struct Message {
        address sender;
        bytes message;
        uint256 timestamp;
        uint256 blockNumber;
    }

    mapping (uint256 => Message) public messages;
    uint256 public totalMessages;

    function onMessageReceived(
        address _sender,
        bytes memory _message,
        uint256 _timestamp,
        uint256 _blockNumber
    )
        internal
    {
        messages[totalMessages] = Message({
            sender: _sender,
            message: _message,
            timestamp: _timestamp,
            blockNumber: _blockNumber
        });
        totalMessages += 1;
    }
}