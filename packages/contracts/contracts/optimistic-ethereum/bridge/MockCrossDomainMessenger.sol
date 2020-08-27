pragma solidity ^0.5.0;

/* Interface Imports */
import { ICrossDomainMessenger } from "./CrossDomainMessenger.interface.sol";

/**
 * @title MockCrossDomainMessenger
 */
contract MockCrossDomainMessenger is ICrossDomainMessenger {
    /*
     * Contract Variables
     */

    ICrossDomainMessenger targetMessenger;
    address public crossDomainMsgSender;

    /*
     * Public Functions
     */

    /**
     * Relays a message to a target contract.
     * .inheritdoc ICrossDomainMessenger
     */
    function relayMessage(
        address _target,
        address _sender,
        bytes memory _message
    )
        public
    {
        crossDomainMsgSender = _sender;
        (bool success,) = _target.call(_message);
        require(success, "Received message reverted during execution.");
    }

    /**
     * Sends a message to the target messenger.
     * .inheritdoc ICrossDomainMessenger
     */
    function sendMessage(
        address _target,
        bytes memory _message
    )
        public
    {
        require(
            address(targetMessenger) != address(0),
            "Cannot send a message without setting the target messenger."
        );

        targetMessenger.relayMessage(
            _target,
            msg.sender,
            _message
        );
    }

    /**
     * Sets the target messenger.
     * @param _messenger Target messenger address.
     */
    function setTargetMessenger(
        address _messenger
    )
        public
    {
        targetMessenger = ICrossDomainMessenger(_messenger);
    }
}
