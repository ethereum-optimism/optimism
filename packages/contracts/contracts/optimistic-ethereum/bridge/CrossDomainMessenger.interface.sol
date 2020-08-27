pragma solidity ^0.5.0;

contract ICrossDomainMessenger {
    address public crossDomainMsgSender;

    /**
     * Relays a message to a given target contract.
     * @param _target Address of the target contract.
     * @param _sender Address of the message sender.
     * @param _message Calldata to relay.
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message
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