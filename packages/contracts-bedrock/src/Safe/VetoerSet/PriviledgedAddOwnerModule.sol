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
    Safe internal immutable safe;

    /// @notice The OwnerGuard instance
    OwnerGuard internal immutable ownerGuard;

    /// @notice The OP Foundation multisig address
    address internal immutable opFoundation;

    /// @notice Thrown when trying to add an owner through this module but the caller is not
    ///         the whitelisted OP Foundation address.
    /// @param sender The sender address.
    error SenderIsNotOpFoundation(address sender);

    /// @notice The module constructor.
    /// @param safe_ The Safe wallet address
    /// @param ownerGuard_ The owner guard contract address.
    /// @param opFoundation_ The OP Foundation multisig address.
    constructor(Safe safe_, OwnerGuard ownerGuard_, address opFoundation_) {
        safe = safe_;
        ownerGuard = ownerGuard_;
        opFoundation = opFoundation_;
    }

    /// @notice Add a new owner address.
    /// @dev Revert if not called by the whitelised `opFoundation` address.
    /// @param addr The owner address to add.
    function priviledgedAddOwner(address addr) external {
        // Ensure the caller is the OP Foundation multisig.
        if (msg.sender != opFoundation) {
            revert SenderIsNotOpFoundation(msg.sender);
        }

        // Ensure adding a new owner is possible (i.e. the `maxCount` is not exceeded).
        uint256 threshold = ownerGuard.checkNewOwnerCount(safe.getOwners().length + 1);

        // Add a new owner to the Safe wallet, specifying the new threshold.
        safe.execTransactionFromModule(
            address(safe), 0, abi.encodeCall(safe.addOwnerWithThreshold, (addr, threshold)), Enum.Operation.Call
        );
    }
}
