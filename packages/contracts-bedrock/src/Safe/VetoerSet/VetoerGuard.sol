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
    uint8 internal maxCount = 7;

    /// @notice The safe account for which this contract will be the guard.
    Safe internal immutable SAFE;

    /// @notice Constructor.
    /// @param _safe The safe account for which this contract will be the guard.
    constructor(Safe _safe) {
        SAFE = _safe;
    }

    function checkAfterExecution(bytes32, bool) external {
        _requireOnlySafe();
        // Get the current set of owners
        address[] memory ownersAfter = SAFE.getOwners();

        // TODO: require owners to be less than or equal to maxCount
        // TODO: require threshold to owners ratio to equal 66%
    }

    function increaseMaxCount(uint8 _newCount) external {
    // TODO: add function to increase maxCount, authed on Safe itself
    }

    /// @notice Getter function for the Safe contract instance
    /// @return safe_ The Safe contract instance
    function safe() public view returns (Safe safe_) {
        safe_ = SAFE;
    }

    /// @notice Internal function to ensure that only the Safe can call certain functions.
    function _requireOnlySafe() internal view {
        require(msg.sender == address(SAFE), "VetoerGuard: only Safe can call this function");
    }
}
