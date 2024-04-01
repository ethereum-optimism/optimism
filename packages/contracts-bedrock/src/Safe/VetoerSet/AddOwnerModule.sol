// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

import { ISemver } from "src/universal/ISemver.sol";

import { OwnerGuard } from "./OwnerGuard.sol";

/// @title AddOwnerModule
/// @notice This module allows any specifically designated address to add owners to the
///         Safe Account. Specifically, the Optimism Foundation may add new owners.
contract AddOwnerModule is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe contract instance
    Safe public immutable safe;

    /// @notice The OwnerGuard instance
    OwnerGuard public immutable ownerGuard;

    /// @notice The admin address, allowed to add new owners.
    address public immutable admin;

    /// @notice Thrown when trying to add an owner through this module but the caller is not
    ///         the registered admin address.
    /// @param sender The sender address.
    error SenderIsNotAdmin(address sender);

    /// @notice The module constructor.
    /// @param _safe The Safe Account address
    /// @param _ownerGuard The owner guard contract address.
    /// @param _admin The admin address.
    constructor(Safe _safe, OwnerGuard _ownerGuard, address _admin) {
        safe = _safe;
        ownerGuard = _ownerGuard;
        admin = _admin;
    }

    /// @notice Add a new owner address.
    /// @dev Revert if not called by the whitelisted `admin` address.
    /// @param addr The owner address to add.
    function addOwner(address addr) external {
        // Ensure the caller is the registered admin.
        if (msg.sender != admin) {
            revert SenderIsNotAdmin(msg.sender);
        }

        // Ensure adding a new owner is possible (i.e. the `maxCount` is not exceeded).
        uint256 threshold = ownerGuard.checkNewOwnerCount(safe.getOwners().length + 1);

        // Add a new owner to the Safe Account, specifying the new threshold.
        safe.execTransactionFromModule(
            address(safe), 0, abi.encodeCall(safe.addOwnerWithThreshold, (addr, threshold)), Enum.Operation.Call
        );
    }
}
