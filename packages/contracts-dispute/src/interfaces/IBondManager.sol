// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title IBondManager
/// @notice The Bond Manager holds ether posted as a bond for a bond id.
interface IBondManager {
    /// @notice Post a bond for a given id.
    function post(bytes32 id) external payable;

    /// @notice Calls a bond for a given bond id.
    /// @notice Only the address that posted the bond may claim it.
    function call(bytes32 id, address to) external returns (uint256);

    /// @notice Returns the next minimum bond amount.
    function next() external returns (uint256);
}
