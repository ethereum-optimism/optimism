// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum } from "safe-contracts/Safe.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title VetoModule
/// @notice This module allows any owner of the Safe Account to execute a veto.
contract VetoModule is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe contract instance
    Safe public immutable safe;

    /// @notice The delayed vetoable contract.
    address public immutable delayedVetoable;

    /// @notice Thrown when trying to execute a veto through this module but the caller is not
    ///         an owner of the Safe Account.
    /// @param sender The sender address.
    error SenderIsNotAnOwner(address sender);

    /// @notice The module constructor.
    /// @param _safe The Safe Account addess.
    /// @param _delayedVetoable The delay vetoable contract address.
    constructor(Safe _safe, address _delayedVetoable) {
        safe = _safe;
        delayedVetoable = _delayedVetoable;
    }

    /// @notice Passthrough for any owner to execute a veto on the `DelayedVetoable` contract.
    /// @dev Revert if not called by an owner of the Safe Account.
    function veto() external returns (bool) {
        // Ensure only a Safe Account owner can veto.
        if (safe.isOwner(msg.sender) == false) {
            revert SenderIsNotAnOwner(msg.sender);
        }

        // Forward the call to the Safe Account, targeting the delayed vetoable contract.
        return safe.execTransactionFromModule(delayedVetoable, 0, msg.data, Enum.Operation.Call);
    }
}
