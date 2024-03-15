// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

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

    /// @notice The module constructor.
    /// @param _safe The Safe wallet address
    /// @param _vetoerGuard The vetoer guard contract address.
    /// @param _op The OP Foundation multisig address.
    constructor(Safe _safe, VetoerGuard _vetoerGuard, address _op) {
        SAFE = _safe;
        VETOER_GUARD = _vetoerGuard;
        OP_FOUNDATION = _op;
    }

    /// @notice Add a new vetoer address.
    /// @dev Revert if not called by the whitelised `OP_FOUNDATION` address.
    /// @param _addr The vetoer address to add.
    function addVetoer(address _addr) external {
        // Ensure the caller is the OP Foundation multisig.
        require(msg.sender == OP_FOUNDATION, "AddVetoerModule: only OP Foundation can call addVetoer");

        // Ensure adding a new vetoer is possible (i.e. the `maxCount` is not exceeded).
        uint256 threshold = VETOER_GUARD.checkNewOwnerCount(SAFE.getOwners().length + 1);

        // Add a new owner to the Safe wallet, specifying the new threshold.
        SAFE.execTransactionFromModule(
            address(SAFE), 0, abi.encodeCall(SAFE.addOwnerWithThreshold, (_addr, threshold)), Enum.Operation.Call
        );
    }
}
