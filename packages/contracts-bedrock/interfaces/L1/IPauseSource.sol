// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";

/// @title IPauseSource
/// @notice Interface for a scoped pause source.
interface IPauseSource {
    /// @notice Returns whether this pause source is paused.
    /// @return Whether the system is paused.
    function paused() external view returns (bool);

    /// @notice Returns the guardian.
    /// @return The guardian address.
    function guardian() external view returns (address);

    /// @notice Returns the SuperchainConfig.
    /// @return The SuperchainConfig contract.
    function superchainConfig() external view returns (ISuperchainConfig);
}
