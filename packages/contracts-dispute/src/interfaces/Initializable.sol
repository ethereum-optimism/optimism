// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

/// @title Initializable
/// @notice An interface for initializable contracts.
interface Initializable {
    /// @notice Initializes the contract.
    /// @custom:invariant The `initialize` function may only be called once.
    function initialize() external;
}
