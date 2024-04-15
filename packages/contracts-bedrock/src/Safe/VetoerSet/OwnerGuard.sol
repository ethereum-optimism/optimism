// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { FixedPointMathLib } from "solady/utils/FixedPointMathLib.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title OwnerGuard
/// @notice This Guard contract is used to enforce a maximum number of owners and a required
///         threshold for the Safe Account.
contract OwnerGuard is ISemver, BaseGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The initial max owner count used at deployment.
    uint256 public constant INITIAL_MAX_OWNER_COUNT = 7;

    /// @notice The Safe Account for which this contract will be the guard.
    Safe public immutable safe;

    /// @notice The maximum number of owners. Can be changed by supermajority.
    uint256 public maxOwnerCount;

    /// @notice Thrown if the new owner count is above the `maxOwnerCount` limit.
    /// @param ownerCount The Safe Account owner count.
    /// @param maxOwnerCount The current `maxOwnerCount`.
    error OwnerCountTooHigh(uint256 ownerCount, uint256 maxOwnerCount);

    /// @notice Thrown after the Safe Account executed a transaction if its threshold does not matches
    ///         with the desired 66% threshold.
    /// @param threshold The Safe Account threshold.
    /// @param expectedThreshold The expected threshold.
    error InvalidSafeAccountThreshold(uint256 threshold, uint256 expectedThreshold);

    /// @notice Thrown when trying to update the `maxOwnerCount` limit but the caller is not
    ///         the associated Safe Account.
    /// @param sender The sender address.
    error SenderIsNotSafeAccount(address sender);

    /// @notice Thrown when trying to update the `maxOwnerCount` limit to a value lower than the current
    ///         Safe Account owner count.
    /// @param newMaxOwnerCount The new invalid max count.
    /// @param ownerCount The current Safe Account owner count.
    error MaxOwnerCountTooLow(uint256 newMaxOwnerCount, uint256 ownerCount);

    /// @notice Constructor.
    /// @param _safe The Safe Account for which this contract will be the guard.
    constructor(Safe _safe) {
        safe = _safe;

        // Set the initial `maxOwnerCount`, to the greater between `INITIAL_MAX_OWNER_COUNT` and the current owner
        // count.
        maxOwnerCount = FixedPointMathLib.max(INITIAL_MAX_OWNER_COUNT, _safe.getOwners().length);
    }

    /// @notice Inherited hook from the `Guard` interface that is run right before the transaction is executed
    ///         by the Safe Account when `execTransaction` is called.
    /// @dev All checks are performed in `checkAfterExecution()` so this method is left empty.
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

    /// @notice Inherited hook from the `Guard` interface that is run right after the transaction has been executed
    ///         by the Safe Account when `execTransaction` is called.
    /// @dev Reverts if the Safe Account owner count is above the limit specified by `maxOwnerCount`.
    /// @dev Reverts if the Safe Account threshold is not equal to the expected 66% threshold.
    function checkAfterExecution(bytes32, bool) external view {
        // Ensure the length of the new set of owners is not above `maxOwnerCount`, and get the corresponding threshold.
        uint256 expectedThreshold = checkNewOwnerCount(safe.getOwners().length);

        // Ensure the Safe Account threshold always stays in sync with the 66% one.
        uint256 threshold = safe.getThreshold();
        if (threshold != expectedThreshold) {
            revert InvalidSafeAccountThreshold(threshold, expectedThreshold);
        }
    }

    /// @notice Update the maximum number of owners.
    /// @dev Reverts if not called by the Safe Account.
    /// @param newMaxOwnerCount The new possible `maxOwnerCount` of owners.
    function updateMaxOwnerCount(uint256 newMaxOwnerCount) external {
        // Ensure only the Safe Account can call this function.
        if (msg.sender != address(safe)) {
            revert SenderIsNotSafeAccount(msg.sender);
        }

        // Ensure the given `newMaxOwnerCount` is not bellow the current number of owners.
        uint256 ownerCount = safe.getOwners().length;
        if (newMaxOwnerCount < ownerCount) {
            revert MaxOwnerCountTooLow(newMaxOwnerCount, ownerCount);
        }

        // Update the new `maxOwnerCount`.
        maxOwnerCount = newMaxOwnerCount;
    }

    /// @notice Checks if the given `newOwnerCount` of owners is allowed and returns the corresponding 66% threshold.
    /// @dev Reverts if `newOwnerCount` is above `maxOwnerCount`.
    /// @param newOwnerCount The owners count to check.
    /// @return threshold The corresponding 66% threshold for `newOwnerCount` owners.
    function checkNewOwnerCount(uint256 newOwnerCount) public view returns (uint256 threshold) {
        // Ensure we don't exceed the maximum number of allowed owners.
        if (newOwnerCount > maxOwnerCount) {
            revert OwnerCountTooHigh(newOwnerCount, maxOwnerCount);
        }

        // Compute the corresponding ceil (66%) threshold of owners.
        threshold = (newOwnerCount * 66 + 99) / 100;
    }
}
