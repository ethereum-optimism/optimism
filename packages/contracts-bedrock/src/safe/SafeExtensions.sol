// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe Extensions
import { LivenessModule2 } from "./LivenessModule2.sol";
import { TimelockGuard } from "./TimelockGuard.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title SafeExtensions
/// @notice Combined Safe extensions providing both liveness module and timelock guard functionality
/// @dev This contract can be enabled simultaneously as both a module and a guard on a Safe:
///      - As a module: provides liveness challenge functionality to prevent multisig deadlock
///      - As a guard: provides timelock functionality for transaction delays and cancellation
contract SafeExtensions is LivenessModule2, TimelockGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";
}
