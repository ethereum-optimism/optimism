// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/// @title IOwnable
/// @notice An interface for ownable contracts.
interface IOwnable {
    /// @notice Returns the owner of the contract
    function owner() external view returns (address);

    /// @notice Transfer ownership to the passed address
    /// @param newOwner The address to transfer ownership to
    /// @dev May only be called by the `owner`.
    function transferOwnership(address newOwner) external;
}
