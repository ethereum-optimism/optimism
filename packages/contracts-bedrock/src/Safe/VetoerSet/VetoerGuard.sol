// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { BaseGuard } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title VetoerSetGuard
/// @notice This Guard contract is used to enforce a specific threshold for the VetoerSet.
contract VetoerGuard is ISemver, BaseGuard {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The maximum number of vetoers in the VetoerSet. May be changed by supermajority.
    uint8 public maxCount = 7;

    /// @notice The safe account for which this contract will be the guard.
    Safe internal immutable SAFE;

    /// @notice Constructor.
    /// @param _safe The safe account for which this contract will be the guard.
    constructor(Safe _safe) {
        SAFE = _safe;
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
        // Ensure the length of the new set of owners is not above `maxCount`, and get the corresponding threshold.
        uint256 threshold_ = checkNewOwnerCount(SAFE.getOwners().length);

        // Ensure the Safe Wallet threshold always stays in sync with the 66% one.
        require(
            SAFE.getThreshold() == threshold_,
            "VetoerGuard: Safe must have a threshold of at least 66% of the number of owners"
        );
    }

    /// @notice Checks if the given `_newCount` of vetoers is allowed and returns the corresponding 66% threshold.
    /// @dev Reverts if `_newCount` is above `maxCount`.
    /// @param _newCount The vetoers count to check.
    /// @return threshold_ The corresponding 66% threshold for `_newCount` vetoers.
    function checkNewOwnerCount(uint256 _newCount) public view returns (uint256 threshold_) {
        // Ensure we don't exceed the maximum number of allowed vetoers.
        require(_newCount <= maxCount, "VetoerGuard: too many owners");

        // Compute the corresponding ceil(66%) threshold of owners.
        threshold_ = (_newCount * 66 + 99) / 100;
    }

    /// @notice Update the maximum number of vetoers.
    /// @dev Reverts if not called by the Safe Wallet.
    /// @param _newMaxCount The new possible `maxCount` of vetoers.
    function updateMaxCount(uint8 _newMaxCount) external {
        // Ensure only the Safe Wallet can call this function.
        require(msg.sender == address(SAFE), "VetoerGuard: only Safe can call this function");

        // Ensure the given `_newMaxCount` is not bellow the current number of owners.
        require(_newMaxCount >= SAFE.getOwners().length);

        // Update the new`maxCount`.
        maxCount = _newMaxCount;
    }

    /// @notice Getter function for the Safe contract instance
    /// @return safe_ The Safe contract instance
    function safe() public view returns (Safe safe_) {
        safe_ = SAFE;
    }
}
