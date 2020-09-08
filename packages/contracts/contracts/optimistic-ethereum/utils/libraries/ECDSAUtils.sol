pragma solidity ^0.5.0;

/**
 * @title ECDSAUtils
 */
library ECDSAUtils {
    /**
     * Recovers a signed address given a message and signature.
     * @param _message Message that was originally signed.
     * @param _isEthSignedMessage Whether or not the user used the `Ethereum Signed Message` prefix.
     * @param _v Signature `v` parameter.
     * @param _r Signature `r` parameter.
     * @param _s Signature `s` parameter.
     * @param _chainId Chain ID parameter.
     * @return Signer address.
     */
    function recover(
        bytes memory _message,
        bool _isEthSignedMessage,
        uint8 _v,
        bytes32 _r,
        bytes32 _s,
        uint256 _chainId
    )
        internal
        pure
        returns (
            address _sender
        )
    {
        bytes32 messageHash = _isEthSignedMessage ? getEthSignedMessageHash(_message) : getNativeMessageHash(_message);
        uint8 v = _isEthSignedMessage ? _v : (_v - uint8(_chainId) * 2) - 8;
        return ecrecover(
            messageHash,
            v,
            _r,
            _s
        );
    }

    /**
     * Gets the native message hash (simple keccak256) for a message.
     * @param _message Message to hash.
     * @return Native message hash.
     */
    function getNativeMessageHash(
        bytes memory _message
    )
        internal
        pure
        returns (
            bytes32 _messageHash
        )
    {
        return keccak256(_message);
    }

    /**
     * Gets the hash of a message with the `Ethereum Signed Message` prefix.
     * @param _message Message to hash.
     * @return Prefixed message hash.
     */
    function getEthSignedMessageHash(
        bytes memory _message
    )
        internal
        pure
        returns (
            bytes32 _messageHash
        )
    {
        bytes memory prefix = "\x19Ethereum Signed Message:\n32";
        bytes32 messageHash = keccak256(_message);
        return keccak256(abi.encodePacked(prefix, messageHash));
    }
}