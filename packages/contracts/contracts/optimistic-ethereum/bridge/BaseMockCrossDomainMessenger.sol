pragma solidity ^0.5.0;

/**
 * @title BaseMockCrossDomainMessenger
 */
contract BaseMockCrossDomainMessenger {
    /*
     * Contract Variables
     */

    mapping (uint256 => bytes) internal fullSentMessages;
    uint256 public messagesToRelay;
    uint256 public lastRelayedMessage;


    /*
     * Public Functions
     */

    function relayMessageToTarget()
        public
    {
        require(
            lastRelayedMessage < messagesToRelay,
            "Already relayed all of the messages!"
        );

        bytes memory topMessage = fullSentMessages[lastRelayedMessage];

        (
            address target,
            address sender,
            bytes memory message,
            uint256 messageNonce
        ) = _decodeXDomainCalldata(topMessage);

        _relayXDomainMessageToTarget(
            target,
            sender,
            message,
            messageNonce
        );

        lastRelayedMessage += 1;
    }


    /*
     * Internal Functions
     */

    /**
     * Sends a cross domain message.
     */
    function _sendXDomainMessage(
        bytes memory _message
    )
        internal
    {
        fullSentMessages[messagesToRelay] = _message;
        messagesToRelay += 1;

        return;
    }

    /**
     * Internal relay function.
     */
    function _relayXDomainMessageToTarget(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
    {
        return;
    }

    /**
     * Generates the correct cross domain calldata for a message.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return ABI encoded cross domain calldata.
     */
    function _decodeXDomainCalldata(
        bytes memory _calldata
    )
        internal
        pure
        returns (
            address _target,
            address _sender,
            bytes memory _message,
            uint256 _messageNonce
        )
    {
        return abi.decode(_calldata, (address, address, bytes, uint256));
    }
}
