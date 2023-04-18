// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title IInitializable
/// @notice An interface for initializable contracts.
interface IInitializable {
    /// @notice Initializes the contract.
    /// @custom:invariant The `initialize` function may only be called once.
    function initialize() external;
}
