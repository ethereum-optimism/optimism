pragma solidity ^0.5.0;

contract ICrossDomainMessenger {
    /**
     * Relays a message to a given target contract.
     * @param _target Address of the target contract.
     * @param _sender Address of the message sender.
     * @param _message Calldata to relay.
     * @param _timestamp Time the message was relayed.
     * @param _blockNumber Block number the message was relayed in.
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _timestamp,
        uint256 _blockNumber
    ) public;

    /**
     * Sends a message to another cross domain messenger to be relayed.
     * @param _target Address of the target contract.
     * @param _message Calldata to relay.
     */
    function sendMessage(
        address _target,
        bytes memory _message
    ) public;
}