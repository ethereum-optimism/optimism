pragma solidity ^0.8.9;

contract MessageEncodingHelper {
    // This function is copy/pasted from the Lib_CrossDomainUtils library. We have to do this
    // because the Lib_CrossDomainUtils library does not provide a function for hashing. Instead,
    // I'm duplicating the functionality of the library here and exposing an additional method that
    // does the required hashing. This is fragile and will break if we ever update the way that our
    // contracts hash the encoded data, but at least it works for now.
    // TODO: Next time we're planning to upgrade the contracts, make sure that the library also
    // contains a function for hashing.
    function encodeXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) public pure returns (bytes memory) {
        return
            abi.encodeWithSignature(
                "relayMessage(address,address,bytes,uint256)",
                _target,
                _sender,
                _message,
                _messageNonce
            );
    }

    function hashXDomainCalldata(
        address _target,
        address _sender,
        bytes memory _message,
        uint256 _messageNonce
    ) public pure returns (bytes32) {
        return keccak256(
            encodeXDomainCalldata(
                _target,
                _sender,
                _message,
                _messageNonce
            )
        );
    }
}
