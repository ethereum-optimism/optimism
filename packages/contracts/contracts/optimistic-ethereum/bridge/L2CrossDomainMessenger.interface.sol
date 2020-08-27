pragma solidity ^0.5.0;

/**
 * @title IL2CrossDomainMessenger
 */
contract IL2CrossDomainMessenger {
    /**
     * Sends a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     */
    function sendMessage(
        address _target,
        bytes memory _message
    ) public;

    /**
     * Relays a cross domain message to a contract.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) public;
}