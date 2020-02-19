pragma solidity ^0.5.0;

contract CallerReturner {
    function getMsgSender() public view returns(address) {
        return msg.sender;
    }
}