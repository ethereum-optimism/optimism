// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title IDelayedVetoable
/// @notice Interface for the DelayedVetoable contract.
interface IDelayedVetoable is ISemver {
    /// @notice Error for when attempting to forward too early.
    error ForwardingEarly();

    /// @notice Error for unauthorized calls.
    error Unauthorized(address expected, address actual);

    /// @notice An event that is emitted when the delay is activated.
    /// @param delay The delay that was activated.
    event DelayActivated(uint256 delay);

    /// @notice An event that is emitted when a call is initiated.
    /// @param callHash The hash of the call data.
    /// @param data The data of the initiated call.
    event Initiated(bytes32 indexed callHash, bytes data);

    /// @notice An event that is emitted each time a call is forwarded.
    /// @param callHash The hash of the call data.
    /// @param data The data forwarded to the target.
    event Forwarded(bytes32 indexed callHash, bytes data);

    /// @notice An event that is emitted each time a call is vetoed.
    /// @param callHash The hash of the call data.
    /// @param data The data forwarded to the target.
    event Vetoed(bytes32 indexed callHash, bytes data);

    function initiator() external returns (address initiator_);
    function vetoer() external returns (address vetoer_);
    function target() external returns (address target_);
    function delay() external returns (uint256 delay_);
    function queuedAt(bytes32 callHash) external returns (uint256 queuedAt_);
    fallback() external;
}
