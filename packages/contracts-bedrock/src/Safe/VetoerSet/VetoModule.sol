// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Safe } from "safe-contracts/Safe.sol";
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

    constructor(Safe _safe, address _delayedVetoable) {
        SAFE = _safe;
    }

    /// @notice Passthrough for any vetoer to execute a veto on the DelayedVetoable contract
    function veto() external returns (bool) {
        require(SAFE.isOwner(msg.sender), "VetoModule: only vetoers can call veto");

        // assembly can't use immutable value, so creating local variable for ease
        address delayedVetoable = address(DELAYED_VETOABLE);

        // TODO: execute a transaction as the Safe (i.e. SAFE.execTransactionFromModule())
    }
}
