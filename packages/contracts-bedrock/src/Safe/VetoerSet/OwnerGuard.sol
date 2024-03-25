// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { FixedPointMathLib } from "solady/utils/FixedPointMathLib.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title OwnerGuard
/// @notice This Guard contract is used to enforce a maximum number of owners and a required
///         threshold for the Safe Wallet.
contract OwnerGuard is ISemver, BaseGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe Wallet for which this contract will be the guard.
    Safe public immutable safe;

    /// @notice The maximum number of owners. Can be changed by supermajority.
    uint8 public maxOwnerCount;

    /// @notice Thrown at deployment if the current Safe Wallet owner count can't fit in a `uint8`.
    /// @param ownerCount The current owner count.
    error OwnerCountTooHigh(uint256 ownerCount);

    /// @notice Thrown if the new owner count is above the `maxOwnerCount` limit.
    /// @param ownerCount The Safe Wallet owner count.
    /// @param maxOwerCount The current `maxOwnerCount`.
    error InvalidOwnerCount(uint256 ownerCount, uint256 maxOwerCount);

    /// @notice Thrown after the Safe Wallet executed a transaction if its threshold does not matches
    ///         with the desired 66% threshold.
    /// @param threshold The Safe Wallet threshold.
    /// @param expectedThreshold The expected threshold.
    error InvalidSafeWalletThreshold(uint256 threshold, uint256 expectedThreshold);

    /// @notice Thrown when trying to update the `maxOwnerCount` limit but the caller is not
    ///         the associated Safe Wallet.
    /// @param sender The sender address.
    error SenderIsNotSafeWallet(address sender);

    /// @notice Thrown when trying to update the `maxOwnerCount` limit to a value lower than the current
    ///         Safe Wallet owner count.
    /// @param newMaxOwnerCount The new invalid max count.
    /// @param ownerCount The current Safe Wallet owner count.
    error InvalidNewMaxCount(uint256 newMaxOwnerCount, uint256 ownerCount);

    /// @notice Constructor.
    /// @param safe_ The Safe Wallet for which this contract will be the guard.
    constructor(Safe safe_) {
        safe = safe_;

        // Get the current owner count of the Smart Wallet.
        uint256 ownerCount = safe_.getOwners().length;
        if (ownerCount > type(uint8).max) {
            revert OwnerCountTooHigh(ownerCount);
        }

        // Set the initial `maxOwnerCount`, to the greater between 7 and the current owner count.
        maxOwnerCount = uint8(FixedPointMathLib.max(7, ownerCount));
    }

    /// @notice Inherited hook from `BaseGuard` that is run right before the transaction is executed
    ///         by the Safe Wallet when `execTransaction` is called.
    function checkTransaction(
        address,
        uint256,
        bytes memory,
        Enum.Operation,
        uint256,
        uint256,
        uint256,
        address,
        address payable,
        bytes memory,
        address
    )
        external
    { }

    /// @notice Inherited hook from `BaseGuard` that is run right after the transaction has been executed
    ///         by the Safe Wallet when `execTransaction` is called.
    function checkAfterExecution(bytes32, bool) external view {
        // Ensure the length of the new set of owners is not above `maxOwnerCount`, and get the corresponding threshold.
        uint256 expectedThreshold = checkNewOwnerCount(safe.getOwners().length);

        // Ensure the Safe Wallet threshold always stays in sync with the 66% one.
        uint256 threshold = safe.getThreshold();
        if (threshold != expectedThreshold) {
            revert InvalidSafeWalletThreshold(threshold, expectedThreshold);
        }
    }

    /// @notice Update the maximum number of owners.
    /// @dev Reverts if not called by the Safe Wallet.
    /// @param newMaxOwnerCount The new possible `maxOwnerCount` of owners.
    function updateMaxCount(uint8 newMaxOwnerCount) external {
        // Ensure only the Safe Wallet can call this function.
        if (msg.sender != address(safe)) {
            revert SenderIsNotSafeWallet(msg.sender);
        }

        // Ensure the given `newMaxOwnerCount` is not bellow the current number of owners.
        uint256 ownerCount = safe.getOwners().length;
        if (newMaxOwnerCount < ownerCount) {
            revert InvalidNewMaxCount(newMaxOwnerCount, ownerCount);
        }

        // Update the new`maxOwnerCount`.
        maxOwnerCount = newMaxOwnerCount;
    }

    /// @notice Checks if the given `newOwnerCount` of owners is allowed and returns the corresponding 66% threshold.
    /// @dev Reverts if `newOwnerCount` is above `maxOwnerCount`.
    /// @param newOwnerCount The owners count to check.
    /// @return threshold The corresponding 66% threshold for `newOwnerCount` owners.
    function checkNewOwnerCount(uint256 newOwnerCount) public view returns (uint256 threshold) {
        // Ensure we don't exceed the maximum number of allowed owners.
        if (newOwnerCount > maxOwnerCount) {
            revert InvalidOwnerCount(newOwnerCount, maxOwnerCount);
        }

        // Compute the corresponding ceil(66%) threshold of owners.
        threshold = (newOwnerCount * 66 + 99) / 100;
    }
}
