// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { Enum } from "safe-contracts/common/Enum.sol";

// Safe Extensions
import { LivenessModule2 } from "./LivenessModule2.sol";
import { TimelockGuard } from "./TimelockGuard.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

// Libraries
import { EnumerableSet } from "@openzeppelin/contracts/utils/structs/EnumerableSet.sol";

/// @title SaferSafes
/// @notice Combined Safe extensions providing both liveness module and timelock guard functionality
/// @dev This contract can be enabled simultaneously as both a module and a guard on a Safe:
///      - As a module: provides liveness challenge functionality to prevent multisig deadlock
///      - As a guard: provides timelock functionality for transaction delays and cancellation
///      The two components in this contract are almost entirely independent of each other, and can be treated as
///      separate extensions to the Safe. The only shared logic is the _checkCombinedConfig which runs at the end of the
///      configuration functions for both components and ensures that the resulting configuration is valid.
///      Either component can be enabled or disabled independently of the other.
///      When installing either component, it should first be enabled, and then configured. If a component's
///      functionality is not desired, then there is no need to enable or configure it.
contract SaferSafes is LivenessModule2, TimelockGuard, ISemver {
    using EnumerableSet for EnumerableSet.Bytes32Set;

    /// @notice Semantic version.
    /// @custom:semver 1.1.0
    string public constant version = "1.1.0";

    /// @notice Error for when the liveness response period is insufficient.
    error SaferSafes_InsufficientLivenessResponsePeriod();

    /// @notice Internal helper function which can be overriden in a child contract to check if the guard's
    ///         configuration is valid in the context of other extensions that are enabled on the Safe.
    ///         This function acts as a FREI-PI invariant check to ensure the resulting config is valid, it MUST be
    ///         called at the end of any configuration functions in the parent contract.
    function _checkCombinedConfig(Safe _safe) internal view override(LivenessModule2, TimelockGuard) {
        // We only need to perform this check if both the guard and the module are enabled on the Safe
        if (!(_isGuardEnabled(_safe) && _safe.isModuleEnabled(address(this)))) {
            return;
        }

        uint256 timelockDelay = _safeState[_safe].timelockDelay;
        uint256 livenessResponsePeriod = livenessSafeConfiguration[address(_safe)].livenessResponsePeriod;

        // If the timelock delay is 0, then the timelock guard is enabled but not configured.
        // No delay is applied to transactions, so we don't need to perform any further checks.
        if (timelockDelay == 0) {
            return;
        }

        // If the liveness response period is 0, then the liveness module is enabled but not configured.
        // Challenging is not possible, so we don't need to perform any further checks.
        if (livenessResponsePeriod == 0) {
            return;
        }

        // The liveness response period must be at least twice the timelock delay, this is necessary to prevent a
        // situation in which a Safe is not able to respond because there is insufficient time to respond to a challenge
        // after the timelock delay has expired.
        if (livenessResponsePeriod < 2 * timelockDelay) {
            revert SaferSafes_InsufficientLivenessResponsePeriod();
        }
    }

    // TODO: should this be moved into the TimelockGuard contract?
    // Or should this be exposed a public function "clearTimelockGuard" function?
    /// @notice Internal function to disable the guard from the given Safe.
    /// @dev This function is intended for use in the SaferSafes contract, which extends this contract.
    /// @param _targetSafe The Safe instance to disable the guard from.
    function _disableGuard(Safe _targetSafe) internal override {
        SafeState storage safeState = _safeState[_targetSafe];
        // set the timelock delay to 0 to clear the configuration
        safeState.timelockDelay = 0;

        // Reset the cancellation threshold, 1 is the default value for all safes.
        safeState.cancellationThreshold = 0;

        // Get all pending transaction hashes
        bytes32[] memory hashes = safeState.pendingTxHashes.values();

        // Cancel all pending transactions
        // It is true that iterating over a very large array can lead to gas issues, however the number of pending
        // transactions is not expected to be large. If it grows to a point where this becomes an issue, then it maybe
        // be necessary to manually cancel enough transactions to reduce the array size to a manageable size.
        for (uint256 i = 0; i < hashes.length; i++) {
            safeState.pendingTxHashes.remove(hashes[i]);
            safeState.scheduledTransactions[hashes[i]].state = TransactionState.Cancelled;
            emit TransactionCancelled(_targetSafe, hashes[i]);
        }

        // Disable the guard
        // Note that this will remove whichever guard is currently set on the Safe,
        // even if it is not the SaferSafes guard. This is intentional, as it is possible that the guard
        // itself was the cause of the liveness failure which resulted in the transfer of ownership to
        // the fallback owner.
        _targetSafe.execTransactionFromModule({
            to: address(_targetSafe),
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(GuardManager.setGuard, (address(0)))
        });
    }
}
