pragma solidity ^0.5.0;

library ECDSAUtils {
    function recoverNative(
        bytes memory _message,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        internal
        pure
        returns (
            address _sender
        )
    {
        bytes32 messageHash = keccak256(_message);
        return ecrecover(
            messageHash,
            _v,
            _r,
            _s
        );
    }

    function recoverEthSignedMessage(
        bytes memory _message,
        uint8 _v,
        bytes32 _r,
        bytes32 _s
    )
        internal
        pure
        returns (
            address _sender
        )
    {
        bytes memory prefix = "\x19Ethereum Signed Message:\n32";
        bytes32 messageHash = keccak256(_message);
        bytes32 prefixedHash = keccak256(abi.encodePacked(prefix, messageHash));
        return ecrecover(
            prefixedHash,
            _v, 
            _r,
            _s
        );
    }
}