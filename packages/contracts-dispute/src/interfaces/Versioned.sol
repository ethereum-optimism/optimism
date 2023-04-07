// SPDX-License-Identifier: MIT
pragma solidity ^0.8.17;

/// @title Versioned
/// @notice An interface for semantically versioned contracts.
interface Versioned {
    /// @notice Returns the semantic version of the contract
    function version() external pure returns (string memory _version);
}
