pragma solidity ^0.5.0;

/* Interface Imports */
import { ICrossDomainMessageReceiver } from "./CrossDomainMessageReceiver.interface.sol";
import { ICrossDomainMessenger } from "./CrossDomainMessenger.interface.sol";

/**
 * @title MockCrossDomainMessageReceiver
 */
contract MockCrossDomainMessageReceiver is ICrossDomainMessageReceiver {
    /*
     * Contract Variables
     */

    ICrossDomainMessenger public crossDomainMessenger;


    /*
     * Modifiers
     */

    modifier onlyMessenger() {
        require(
            msg.sender == address(crossDomainMessenger),
            "Only the CrossDomainMessenger can call this function."
        );
        _;
    }

    
    /*
     * Public Functions
     */

    /**
     * Receives a message from the cross domain messenger.
     * .inheritdoc ICrossDomainMessageReceiver
     */
    function receiveMessage(
        address _sender,
        bytes memory _message,
        uint256 _timestamp,
        uint256 _blockNumber
    )
        public
        onlyMessenger
    {
        onMessageReceived(
            _sender,
            _message,
            _timestamp,
            _blockNumber
        );

        uint256 messageSize = _message.length;
        bool success = false;
        assembly {
            success := call(
                gas,
                address,
                0,
                add(_message, 0x20),
                messageSize,
                0,
                0
            )
        }

        if (!success) {
            revert("Received message reverted during execution.");
        }
    }

    /**
     * Sets the address of the cross domain messenger.
     * @param _crossDomainMessenger Cross domain messenger address.
     */
    function setMessenger(
        address _crossDomainMessenger
    )
        public
    {
        crossDomainMessenger = ICrossDomainMessenger(_crossDomainMessenger);
    }


    /*
     * Internal Functions
     */

    /**
     * Triggered whenever a message is received.
     * @param _sender Address of the message sender.
     * @param _message Calldata being received.
     * @param _timestamp Time the message was sent.
     * @param _blockNumber Block the message was sent in.
     */
    function onMessageReceived(
        address _sender,
        bytes memory _message,
        uint256 _timestamp,
        uint256 _blockNumber
    )
        internal
    {
        // Implement me!
        return;
    }
}
