pragma solidity ^0.5.16;

contract TimestampChecker {
    uint timestamp;

    function blockTimestamp() public view returns (uint) {
        return block.timestamp;
    }

    function getTimestamp() public view returns (uint) {
        return timestamp;
    }

    function setTimestamp() public {
        timestamp = getTimestamp();
    }
}
