// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum } from "safe-contracts/Safe.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title VetoModule
/// @notice This module allows any owner of the Safe Wallet to execute a veto through this.
contract VetoModule is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe contract instance
    Safe internal immutable safe;

    /// @notice The delayed vetoable contract.
    address internal immutable delayedVetoable;

    /// @notice The module constructor.
    /// @param safe_ The Safe Wallet addess.
    /// @param delayedVetoable_ The delay vetoable contract address.
    constructor(Safe safe_, address delayedVetoable_) {
        safe = safe_;
        delayedVetoable = delayedVetoable_;
    }

    /// @notice Passthrough for any owner to execute a veto on the `DelayedVetoable` contract.
    /// @dev Revert if not called by a Safe Wallet owner address.
    function veto() external returns (bool) {
        // Ensure only a Safe Wallet owner can veto.
        require(safe.isOwner(msg.sender), "VetoModule: only owners can call veto");

        // Forward the call to the Safe Wallet, targeting the delayed vetoable contract.
        return safe.execTransactionFromModule(delayedVetoable, 0, msg.data, Enum.Operation.Call);
    }
}
