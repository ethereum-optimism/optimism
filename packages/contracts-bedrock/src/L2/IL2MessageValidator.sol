// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
/// @title Contract to extend the validation of L1 -> L2 messages on L2.

interface IL2MessageValidator {
    /// @notice Returns a boolean indicating if the message passes additional validation.
    /// @param _nonce       Nonce of the message being relayed.
    /// @param _sender      Address of the user who sent the message.
    /// @param _target      Address that the message is targeted at.
    /// @param _value       ETH value to send with the message.
    /// @param _message     Message to send to the target.
    /// @return bool
    function validateMessage(
        uint256 _nonce,
        address _sender,
        address _target,
        uint256 _value,
        bytes calldata _message
    )
        external
        view
        returns (bool);
}
