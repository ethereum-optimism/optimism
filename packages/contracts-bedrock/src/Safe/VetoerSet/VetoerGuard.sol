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
    ) external {
    }

    function checkAfterExecution(bytes32, bool) external {
        _requireOnlySafe();
        // Get the current set of owners
        address[] memory ownersAfter = SAFE.getOwners();

        uint256 threshold_ = checkNewOwnerCount(ownersAfter.length);
        require(
            SAFE.getThreshold() == threshold_,
            "VetoerGuard: Safe must have a threshold of at least 66% of the number of owners"
        );
    }

    function checkNewOwnerCount(uint256 _newCount) public view returns (uint256 threshold_) {
        require(_newCount <= maxCount, "VetoerGuard: too many owners");
        // require the threshold be ceil(66%) of the owners
        threshold_ = (_newCount * 66 + 99) / 100;
    }

    function increaseMaxCount(uint8 _newCount) external {
        _requireOnlySafe();
        maxCount = _newCount;
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
