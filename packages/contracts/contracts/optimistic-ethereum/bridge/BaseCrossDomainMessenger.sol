pragma solidity ^0.5.0;

/* Interface Imports */
import { ICrossDomainMessenger } from "./interfaces/CrossDomainMessenger.interface.sol";

/**
 * @title BaseCrossDomainMessenger
 */
contract BaseCrossDomainMessenger is ICrossDomainMessenger {

    event SentMessage(bytes32 msgHash);

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
        require(targetMessengerAddress == address(0));
        targetMessengerAddress = _targetMessengerAddress;
    }

    /**
     * Sends a cross domain message to the target messenger.
     * .inheritdoc IL2CrossDomainMessenger
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
        bytes32 msgHash = keccak256(xDomainCalldata);
        sentMessages[msgHash] = true;

        emit SentMessage(msgHash);
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

        /**
     * Sends a cross domain message.
     * @param _message Message to send.
     * @param _gasLimit OVM gas limit for the message.
     */
    function _sendXDomainMessage(
        bytes memory _message,
        uint32 _gasLimit
    ) internal;
}
