// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
/// @title Contract to extend the validation of L1 -> L2 messages on L1.

interface IL1MessageValidator {
    /// @notice Returns a boolean indicating if the message passes additional validation.
    /// @param _data The L1 -> L2 message data.
    /// @param _to   The L1 -> L2 "to" field.
    /// @return bool
    function validateMessage(bytes memory _data, address _to) external view returns (bool);
}
