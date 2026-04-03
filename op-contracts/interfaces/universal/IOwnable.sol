// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IOwnable
/// @notice Interface for Ownable.
interface IOwnable {
    event OwnershipTransferred(address indexed previousOwner, address indexed newOwner);

    /// @dev The caller account is not authorized to perform an operation.
    error OwnableUnauthorizedAccount(address account);

    /// @dev The owner is not a valid owner account. (eg. `address(0)`)
    error OwnableInvalidOwner(address owner);

    function owner() external view returns (address);
    function renounceOwnership() external;
    function transferOwnership(address newOwner) external; // nosemgrep

    function __constructor__() external;
}
