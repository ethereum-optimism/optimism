// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe, Enum } from "safe-contracts/Safe.sol";

import { ISemver } from "src/universal/ISemver.sol";

/// @title ThresholdModule
/// @notice This module is intended to be used in conjunction with the VetoerSetGuard. Any vetoer
///         may execute a veto through this
contract VetoModule is ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice The Safe contract instance
    Safe internal immutable SAFE;

    /// @notice The delayed vetoable contract.
    address internal immutable DELAYED_VETOABLE;

    /// @notice The module constructor.
    /// @param _safe The Safe Wallet addess.
    /// @param _delayedVetoable The delay vetoable contract address.
    constructor(Safe _safe, address _delayedVetoable) {
        SAFE = _safe;
        DELAYED_VETOABLE = _delayedVetoable;
    }

    /// @notice Passthrough for any vetoer to execute a veto on the `DelayedVetoable` contract.
    /// @dev Revert if not called by a Safe Wallet owner address.
    function veto() external returns (bool) {
        // Ensure only a Safe Wallet owner can veto.
        require(SAFE.isOwner(msg.sender), "VetoModule: only vetoers can call veto");

        // Forward the call to the Safe Wallet, targeting the delayed vetoable contract.
        return SAFE.execTransactionFromModule(DELAYED_VETOABLE, 0, msg.data, Enum.Operation.Call);
    }
}
