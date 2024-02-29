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
    address internal immutable DELAYED_VETOABLE;

    constructor(Safe _safe, address _delayedVetoable) {
        SAFE = _safe;
        DELAYED_VETOABLE = _delayedVetoable;
    }

    /// @notice Passthrough for any vetoer to execute a veto on the DelayedVetoable contract
    function veto() external returns (bool) {
        require(SAFE.isOwner(msg.sender), "VetoModule: only vetoers can call veto");

        return SAFE.execTransactionFromModule(
            DELAYED_VETOABLE,
            0,
            msg.data,
            Enum.Operation.Call
        );
    }
}
