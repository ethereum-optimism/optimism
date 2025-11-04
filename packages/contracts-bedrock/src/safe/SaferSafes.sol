// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { Safe } from "safe-contracts/Safe.sol";

// Safe Extensions
import { LivenessModule2 } from "./LivenessModule2.sol";
import { TimelockGuard } from "./TimelockGuard.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title SaferSafes
/// @notice Combined Safe extensions providing both liveness module and timelock guard
///         functionality.
/// @dev This contract can be enabled simultaneously as both a module and a guard on a Safe:
///      - As a module: provides liveness challenge functionality to prevent multisig deadlock
///      - As a guard: provides timelock functionality for transaction delays and cancellation
///      The two components in this contract are almost entirely independent of each other, and can
///      be treated as separate extensions to the Safe. The only shared logic is the
///      _checkCombinedConfig which runs at the end of the configuration functions for both
///      components and ensures that the resulting configuration is valid. Either component can be
///      enabled or disabled independently of the other. When installing either component, it
///      should first be enabled, and then configured. If a component's functionality is not
///      desired, then there is no need to enable or configure it.
///      This contract is compatible only with the Safe contract version 1.4.1 due to the
///      compatibility restrictions in the LivenessModule2 and TimelockGuard contracts.
contract SaferSafes is LivenessModule2, TimelockGuard, ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.9.1
    string public constant version = "1.9.1";

    /// @notice Error for when the liveness response period is insufficient.
    error SaferSafes_InsufficientLivenessResponsePeriod();

    /// @notice Internal helper function which can be overriden in a child contract to check if the
    ///         guard's configuration is valid in the context of other extensions that are enabled
    ///         on the Safe. This function acts as a FREI-PI invariant check to ensure the
    ///         resulting config is valid, it MUST be called at the end of any configuration
    ///         functions in the parent contract.
    function _checkCombinedConfig(Safe _safe) internal view override(LivenessModule2, TimelockGuard) {
        // We only need to perform this check if both the guard and the module are enabled on the
        // Safe
        if (!(_isGuardEnabled(_safe) && _safe.isModuleEnabled(address(this)))) {
            return;
        }

        uint256 timelockDelay = _currentSafeState(_safe).timelockDelay;
        uint256 livenessResponsePeriod = _livenessSafeConfiguration[_safe].livenessResponsePeriod;

        // If the timelock delay is 0, then the timelock guard is enabled but not configured.
        // No delay is applied to transactions, so we don't need to perform any further checks.
        if (timelockDelay == 0) {
            return;
        }

        // If the liveness response period is 0, then the liveness module is enabled but not
        // configured.
        // Challenging is not possible, so we don't need to perform any further checks.
        if (livenessResponsePeriod == 0) {
            return;
        }

        // The liveness response period must be at least twice the timelock delay, this is
        // necessary to prevent a situation in which a Safe is not able to respond because there is
        // insufficient time to respond to a challenge after the timelock delay has expired.
        if (livenessResponsePeriod < 2 * timelockDelay) {
            revert SaferSafes_InsufficientLivenessResponsePeriod();
        }
    }
}
