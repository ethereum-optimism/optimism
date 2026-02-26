// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IDonatable
/// @notice Interface for contracts that accept ETH donations without triggering
///         side effects (deposits on L1, withdrawals on L2). Used by the Compliance
///         module to return ETH to the bridge when a transaction passes all rules
///         during check().
interface IDonatable {
    /// @notice Accepts ETH value without triggering a deposit or withdrawal.
    function donateETH() external payable;
}
