pragma solidity ^0.5.0;

contract L2ToL1MessagePasser {
    function passMessageToL1(bytes memory messageData) public;
}

contract L2ToL1MessageUtil {
    event L2ToL1Message(
        uint _nonce,
        address _ovmSender,
        bytes _callData
    );

    function emitFraudulentMessage() public {
        emit L2ToL1Message(99, address(this), "Great Scott, this is fraudulent!");
    }

    function callMessagePasser(address messagePasserAddress, bytes memory data) public {
        (L2ToL1MessagePasser(messagePasserAddress)).passMessageToL1(data);
    }
}