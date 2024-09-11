// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IL1Block
/// @notice Interface for L1Block with only `isDeposit()` method.
interface IL1Block {
    /// @notice Returns whether the call was triggered from a a deposit or not.
    /// @return True if the current call was triggered by a deposit transaction, and false otherwise.
    function isDeposit() external view returns (bool);
}

