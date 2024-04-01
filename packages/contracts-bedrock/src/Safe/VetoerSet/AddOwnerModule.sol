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

    /// @notice The admin address, allowed to add new owners.
    address public immutable admin;

    /// @notice Storage slot at which the guard address is registered on the Safe Account.
    /// @dev Must match with `StorageAccessible.GUARD_STORAGE_SLOT`.
    bytes32 internal constant GUARD_STORAGE_SLOT = 0x4a204f620c8c5ccdca3fd54d003badd85ba500436a431f0cbda4f558c93c34c8;

    /// @notice Thrown when trying to add an owner through this module but the caller is not
    ///         the registered admin address.
    /// @param sender The sender address.
    error SenderIsNotAdmin(address sender);

    /// @notice The module constructor.
    /// @param _safe The Safe Account address
    /// @param _admin The admin address.
    constructor(Safe _safe, address _admin) {
        safe = _safe;
        admin = _admin;
    }

    /// @notice Add a new owner address.
    /// @dev Revert if not called by the whitelisted `admin` address.
    /// @param addr The owner address to add.
    function addOwner(address addr) external returns (bool) {
        // Ensure the caller is the registered admin.
        if (msg.sender != admin) {
            revert SenderIsNotAdmin(msg.sender);
        }

        // Ensure adding a new owner is possible (i.e. the `maxCount` is not exceeded).
        OwnerGuard ownerGuard = _getOwnerGuard();
        uint256 threshold = ownerGuard.checkNewOwnerCount(safe.getOwners().length + 1);

        // Add a new owner to the Safe Account, specifying the new threshold.
        return safe.execTransactionFromModule(
            address(safe), 0, abi.encodeCall(safe.addOwnerWithThreshold, (addr, threshold)), Enum.Operation.Call
        );
    }

    /// @notice Fetch the `OwnerGuard` address registered on the Safe Account.
    /// @return ownerGuard The `OwnerGuard` address registered on the Safe Account.
    function _getOwnerGuard() internal view returns (OwnerGuard ownerGuard) {
        bytes memory rawBytes = safe.getStorageAt(uint256(GUARD_STORAGE_SLOT), 1);
        assembly ("memory-safe") {
            ownerGuard := mload(add(rawBytes, 32))
        }
    }
}
