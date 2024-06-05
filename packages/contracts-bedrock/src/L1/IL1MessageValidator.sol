// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;
/// @title Contract to extend the validation of L1 -> L2 messages on L1.

interface IL1MessageValidator {
    /// @notice Returns a boolean indicating if the message passes additional validation.
    /// @param _from Msg.sender of the L1 -> L2 message. This address is NOT aliased yet.
    /// @param _to   The L1 -> L2 "to" field.
    /// @param _mint The L1 -> L2 "_mint" field.
    /// @param _value   The L1 -> L2 "_value" field.
    /// @param _gasLimit   The L1 -> L2 "_gasLimit" field.
    /// @return bool
    function validateMessage(
        address _from,
        address _to,
        uint256 _mint,
        uint256 _value,
        uint64 _gasLimit,
        bool _isCreation,
        bytes memory _data
    )
        external
        view
        returns (bool);
}
