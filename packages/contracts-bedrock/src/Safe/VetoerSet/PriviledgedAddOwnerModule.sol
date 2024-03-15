// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { ISemver } from "src/universal/ISemver.sol";

import { OwnerGuard } from "./OwnerGuard.sol";

/// @title PriviledgedAddOwnerModule
/// @notice This module allows any specifically designated address to add owners to the
///         Safe Wallet. Specifically, the Optimism Foundation may add new owners.
contract PriviledgedAddOwnerModule is ISemver {
    /// @notice The Safe contract instance
    Safe internal immutable SAFE;

    /// @notice The OwnerGuard instance
    OwnerGuard internal immutable OWNER_GUARD;

    /// @notice The OP Foundation multisig address
    address internal immutable OP_FOUNDATION;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The module constructor.
    /// @param _safe The Safe wallet address
    /// @param _ownerGuard The owner guard contract address.
    /// @param _op The OP Foundation multisig address.
    constructor(Safe _safe, OwnerGuard _ownerGuard, address _op) {
        SAFE = _safe;
        OWNER_GUARD = _ownerGuard;
        OP_FOUNDATION = _op;
    }

    /// @notice Add a new owner address.
    /// @dev Revert if not called by the whitelised `OP_FOUNDATION` address.
    /// @param _addr The owner address to add.
    function priviledgedAddOwner(address _addr) external {
        // Ensure the caller is the OP Foundation multisig.
        require(msg.sender == OP_FOUNDATION, "PriviledgedAddOwnerModule: only OP Foundation can call addOwner");

        // Ensure adding a new owner is possible (i.e. the `maxCount` is not exceeded).
        uint256 threshold = OWNER_GUARD.checkNewOwnerCount(SAFE.getOwners().length + 1);

        // Add a new owner to the Safe wallet, specifying the new threshold.
        SAFE.execTransactionFromModule(
            address(SAFE), 0, abi.encodeCall(SAFE.addOwnerWithThreshold, (_addr, threshold)), Enum.Operation.Call
        );
    }
}
