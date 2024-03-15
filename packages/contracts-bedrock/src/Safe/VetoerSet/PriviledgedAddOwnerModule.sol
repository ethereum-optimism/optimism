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
     /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe contract instance
    Safe internal immutable SAFE;

    /// @notice The OwnerGuard instance
    OwnerGuard internal immutable OWNER_GUARD;

    /// @notice The OP Foundation multisig address
    address internal immutable OP_FOUNDATION;



    /// @notice The module constructor.
    /// @param safe The Safe wallet address
    /// @param ownerGuard The owner guard contract address.
    /// @param op The OP Foundation multisig address.
    constructor(Safe safe, OwnerGuard ownerGuard, address op) {
        SAFE = safe;
        OWNER_GUARD = ownerGuard;
        OP_FOUNDATION = op;
    }

    /// @notice Add a new owner address.
    /// @dev Revert if not called by the whitelised `OP_FOUNDATION` address.
    /// @param addr The owner address to add.
    function priviledgedAddOwner(address addr) external {
        // Ensure the caller is the OP Foundation multisig.
        require(msg.sender == OP_FOUNDATION, "PriviledgedAddOwnerModule: only OP Foundation can call addOwner");

        // Ensure adding a new owner is possible (i.e. the `maxCount` is not exceeded).
        uint256 threshold = OWNER_GUARD.checkNewOwnerCount(SAFE.getOwners().length + 1);

        // Add a new owner to the Safe wallet, specifying the new threshold.
        SAFE.execTransactionFromModule(
            address(SAFE), 0, abi.encodeCall(SAFE.addOwnerWithThreshold, (addr, threshold)), Enum.Operation.Call
        );
    }
}
