pragma solidity ^0.5.0;

contract BaseMessenger {
    /*
     * Contract Variables
     */

    mapping (bytes32 => bool) receivedMessages;
    mapping (bytes32 => bool) sentMessages;
    address public targetMessengerAddress;
    uint256 public messageNonce;
    address public xDomainMessageSender;


    /*
     * Public Functions
     */

    /**
     * Sets the target messenger on the other domain.
     * @param _targetMessengerAddress Target messenger address.
     */
    function setTargetMessengerAddress(
        address _targetMessengerAddress
    )
        public
    {
        targetMessengerAddress = _targetMessengerAddress;
    }

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
    )
        public
    {
        require(
            _verifyXDomainMessage(
                _target,
                _sender,
                _message,
                _messageNonce
            ) == true,
            "Provided message could not be verified."
        );

        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            _messageNonce
        );

        require(
            receivedMessages[keccak256(xDomainCalldata)] == false,
            "Provided message has already been received."
        );

        xDomainMessageSender = _sender;
        _target.call(_message);

        // Messages are considered successfully executed if they complete
        // without running out of gas (revert or not). As a result, we can
        // ignore the result of the call and always mark the message as
        // successfully executed because we won't get here unless we have
        // enough gas left over.
        receivedMessages[keccak256(xDomainCalldata)] = true;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function sendMessage(
        address _target,
        bytes memory _message,
        uint32 _gasLimit
    )
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            msg.sender,
            _message,
            messageNonce
        );

        _sendXDomainMessage(xDomainCalldata, _gasLimit);

        messageNonce += 1;
        sentMessages[keccak256(xDomainCalldata)] = true;
    }

    /**
     * Replays a cross domain message to the target messenger.
     * @param _target Target contract address.
     * @param _sender Original sender address.
     * @param _message Message to send to the target.
     * @param _gasLimit Gas limit for the provided message.
     */
    function replayMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint32 _gasLimit
    )
        public
    {
        bytes memory xDomainCalldata = _getXDomainCalldata(
            _target,
            _sender,
            _message,
            messageNonce
        );

        require(
            sentMessages[keccak256(xDomainCalldata)] == true,
            "Provided message has not already been sent."
        );

        _sendXDomainMessage(xDomainCalldata, _gasLimit);
    }

    
    /*
     * Internal Functions
     */

    /**
     * Verifies that a received cross domain message is valid.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return whether or not the message is valid.
     */
    function _verifyXDomainMessage(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        internal
        returns (
            bool
        )
    {
        revert("Function must be implemented by children of this contract.");
    }

    /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit OVM gas limit for the message.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint32 _gasLimit
    )
        internal
    {
        revert("Function must be implemented by children of this contract.");
    }


    /*
     * Private Functions
     */

    /**
     * Generates the correct cross domain calldata for a message.
     * @param _target Target contract address.
     * @param _sender Message sender address.
     * @param _message Message to send to the target.
     * @param _messageNonce Nonce for the provided message.
     * @return ABI encoded cross domain calldata.
     */
    function _getXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    )
        private
        view
        returns (
            bytes memory
        )
    {
        return abi.encodeWithSelector(
            bytes4(keccak256(bytes("relayMessage(address,address,bytes,uint256)"))),
            _target,
            _sender,
            _message,
            _messageNonce
        );
    }
}
