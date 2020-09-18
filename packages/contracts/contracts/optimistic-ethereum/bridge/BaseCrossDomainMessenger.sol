pragma solidity ^0.5.0;

/**
 * @title L1CrossDomainMessenger
 */
contract BaseCrossDomainMessenger {
    /*
     * Contract Variables
     */

    mapping (bytes32 => bool) public receivedMessages;
    mapping (bytes32 => bool) public sentMessages;
    address public targetMessengerAddress;
    uint256 public messageNonce;
    address public xDomainMessageSender;


    /*
     * Public Functions
     */

    /**
     * Sets the target messenger address.
     * @param _targetMessengerAddress New messenger address.
     */
    function setTargetMessengerAddress(
        address _targetMessengerAddress
    )
        public
    {
        targetMessengerAddress = _targetMessengerAddress;
    }


    /*
     * Internal Functions
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
        internal
        pure
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