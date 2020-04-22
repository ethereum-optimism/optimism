pragma solidity ^0.5.0;

contract CallerStorer {
    address public lastMsgSender;
    function storeMsgSender() public {
        lastMsgSender = msg.sender;
    }
    function getLastMsgSender() public view returns(address) {
        return lastMsgSender;
    }
}