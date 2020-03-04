pragma solidity ^0.5.0;

contract TimeGetter {
    function getTimestamp() public view returns(uint256) {
        return block.timestamp;
    }
}