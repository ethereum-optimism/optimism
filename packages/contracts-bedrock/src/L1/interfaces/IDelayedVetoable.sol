// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IDelayedVetoable
/// @notice Interface for the DelayedVetoable contract.
interface IDelayedVetoable {
    fallback() external;
    function delay() external returns (uint256 delay_);
    function initiator() external returns (address initiator_);
    function queuedAt(bytes32 callHash) external returns (uint256 queuedAt_);
    function target() external returns (address target_);
    function vetoer() external returns (address vetoer_);
}
