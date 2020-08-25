pragma solidity ^0.5.0;

contract ICrossDomainMessageReceiver {
    /**
     * Receives a message from a cross domain messenger.
     * @param _sender Address of the message sender.
     * @param _message Calldata being received.
     * @param _timestamp Time the message was sent.
     * @param _blockNumber Block the message was sent in.
     */
    function receiveMessage(
        address _sender,
        bytes memory _message,
        uint256 _timestamp,
        uint256 _blockNumber
    ) public;
}