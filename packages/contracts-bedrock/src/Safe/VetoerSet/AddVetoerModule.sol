// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { ISemver } from "src/universal/ISemver.sol";
import { VetoerGuard } from "./VetoerGuard.sol";

/// @title AddVetoerModule
/// @notice This module allows any specifically designated address to add vetoers to the Safe.
///         Specifically, the Optimism Foundation may add vetoers to the VetoerSet.
contract AddVetoerModule is ISemver {
    /// @notice The Safe contract instance
    Safe internal immutable SAFE;

    /// @notice The VetoerGuard instance
    VetoerGuard internal immutable VETOER_GUARD;

    /// @notice The OP Foundation multisig address
    address internal immutable OP_FOUNDATION;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    constructor(Safe _safe, VetoerGuard _vetoerGuard, address _op) {
        SAFE = _safe;
        VETOER_GUARD = _vetoerGuard;
        OP_FOUNDATION = _op;
    }

    function addVetoer(address _addr) external {
        require(msg.sender == OP_FOUNDATION, "AddVetoerModule: only OP Foundation can call addVetoer");
        uint256 threshold = VETOER_GUARD.checkNewOwnerCount(SAFE.getOwners().length + 1);
        SAFE.addOwnerWithThreshold(_addr, threshold);
    }
}
