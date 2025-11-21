// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { Safe } from "safe-contracts/Safe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";

// Libraries
import { SemverComp } from "src/libraries/SemverComp.sol";

/// @title LivenessModule2
/// @notice This module allows challenge-based ownership transfer to a fallback owner
///         when the Safe becomes unresponsive. The fallback owner can initiate a challenge,
///         and if the Safe doesn't respond within the challenge period, ownership transfers
///         to the fallback owner.
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this module using ModuleManager.enableModule()
///      2. The Safe must then configure the module by calling configure() with params
///
///     This guard is compatible only with Safe version 1.4.1.
///
///      Follows a state machine diagram for the lifecycle of this contract:
///      +----------------------+
///      | Start (no challenge) |<---------------------------+
///      +----------------------+                            |
///       |                                                  | respond() by Safe
///       |  challenge() by fallbackOwner                    | OR
///       |                                                  | configureLivenessModule() by Safe
///       v                                                  | OR
///      +--------------------------------------+            | clearLivenessModule() by Safe
///      | Challenge Started                    |            |
///      | challengeStartTime = block.timestamp |------------+
///      +--------------------------------------+            |
///       |                                                  |
///       |  block.timestamp >= challengeStartTime +         |
///       |                     livenessResponsePeriod       |
///       v                                                  |
///      +-----+-----------------------------------------+   |
///      | Ready to transfer ownership to fallback owner |---+
///      +-----+-----------------------------------------+
///       |
///       |  changeOwnershipToFallback() by fallbackOwner
///       |
///       v
///      +------------------------------+
///      | Ownership Transferred        |
///      | - fallback owner sole owner  |
///      | - guard cleared              |
///      | - challenge cleared          |
///      +------------------------------+
///
abstract contract LivenessModule2 {
    /// @notice Configuration for a Safe's liveness module.
    /// @custom:field livenessResponsePeriod The duration in seconds that Safe owners have to
    ///                                      respond to a challenge.
    /// @custom:field fallbackOwner The address that can initiate challenges and claim
    ///                             ownership if the Safe is unresponsive.
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
    }

    /// @notice Mapping from Safe address to its configuration.
    mapping(Safe => ModuleConfig) internal _livenessSafeConfiguration;

    /// @notice Mapping from Safe address to active challenge start time (0 if none).
    mapping(Safe => uint256) public challengeStartTime;

    /// @notice Reserved address used as previous owner to the first owner in a Safe.
    address internal constant SENTINEL_OWNER = address(0x1);

    /// @notice Error for when module is not enabled for the Safe.
    error LivenessModule2_ModuleNotEnabled();

    /// @notice Error for when Safe is not configured for this module.
    error LivenessModule2_ModuleNotConfigured();

    /// @notice Error for when the contract is not 1.4.1.
    error LivenessModule2_InvalidVersion();

    /// @notice Error for when a challenge already exists.
    error LivenessModule2_ChallengeAlreadyExists();

    /// @notice Error for when no challenge exists.
    error LivenessModule2_ChallengeDoesNotExist();

    /// @notice Error for when trying to cancel a challenge after response period has ended.
    error LivenessModule2_ResponsePeriodEnded();

    /// @notice Error for when trying to execute ownership transfer while response period is
    ///         active.
    error LivenessModule2_ResponsePeriodActive();

    /// @notice Error for when caller is not authorized.
    error LivenessModule2_UnauthorizedCaller();

    /// @notice Error for invalid response period.
    error LivenessModule2_InvalidResponsePeriod();

    /// @notice Error for invalid fallback owner.
    error LivenessModule2_InvalidFallbackOwner();

    /// @notice Error for when trying to clear configuration while module is enabled.
    error LivenessModule2_ModuleStillEnabled();

    /// @notice Error for when ownership transfer verification fails.
    error LivenessModule2_OwnershipTransferFailed();

    /// @notice Emitted when a Safe configures the module.
    /// @param safe The Safe address that configured the module.
    /// @param livenessResponsePeriod The duration in seconds that Safe owners have to
    ///                               respond to a challenge.
    /// @param fallbackOwner The address that can initiate challenges and claim ownership if
    ///                      the Safe is unresponsive.
    event ModuleConfigured(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);

    /// @notice Emitted when a Safe clears the module configuration.
    /// @param safe The Safe address that cleared the module configuration.
    event ModuleCleared(address indexed safe);

    /// @notice Emitted when a challenge is started.
    /// @param safe The Safe address that started the challenge.
    /// @param challengeStartTime The timestamp when the challenge started.
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);

    /// @notice Emitted when a challenge is cancelled.
    /// @param safe The Safe address that cancelled the challenge.
    event ChallengeCancelled(address indexed safe);

    /// @notice Emitted when ownership is transferred to the fallback owner.
    /// @param safe The Safe address that succeeded the challenge.
    /// @param fallbackOwner The address that claimed ownership if the Safe is unresponsive.
    event ChallengeSucceeded(address indexed safe, address fallbackOwner);

    ////////////////////////////////////////////////////////////////
    //                   External View Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Returns challenge_start_time + liveness_response_period if challenge exists, or
    ///         0 if not.
    /// @param _safe The Safe address to query.
    /// @return The challenge end timestamp, or 0 if no challenge.
    function getChallengePeriodEnd(Safe _safe) public view returns (uint256) {
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            return 0;
        }
        ModuleConfig storage config = _livenessSafeConfiguration[_safe];
        return startTime + config.livenessResponsePeriod;
    }

    /// @notice Returns the configuration for a given Safe.
    /// @param _safe The Safe address to query.
    /// @return The ModuleConfig for the Safe.
    function livenessSafeConfiguration(Safe _safe) public view returns (ModuleConfig memory) {
        return _livenessSafeConfiguration[_safe];
    }

    ////////////////////////////////////////////////////////////////
    //              External State-Changing Functions             //
    ////////////////////////////////////////////////////////////////

    /// @notice Configures the module for a Safe that has already enabled it.
    /// @param _config The configuration parameters for the module containing the response
    ///                period and fallback owner.
    /// @dev It is strongly recommended that the fallback owner is also a Safe or at least a
    ///      contract that is capable of building and executing transaction batches.
    function configureLivenessModule(ModuleConfig memory _config) external {
        Safe callingSafe = Safe(payable(msg.sender));

        // Validate configuration parameters to ensure module can function properly.
        // livenessResponsePeriod must be > 0 to allow time for Safe owners to respond.
        if (_config.livenessResponsePeriod == 0) {
            revert LivenessModule2_InvalidResponsePeriod();
        }
        // fallbackOwner must not be zero address or the safe itself to be able to become an owner.
        if (_config.fallbackOwner == address(0) || _config.fallbackOwner == address(callingSafe)) {
            revert LivenessModule2_InvalidFallbackOwner();
        }

        // Check that the safe contract version is 1.4.1. There have been breaking changes at every
        // minor version, and we can only support one version.
        if (!SemverComp.eq(callingSafe.VERSION(), "1.4.1")) {
            revert LivenessModule2_InvalidVersion();
        }

        // Check that this module is enabled on the calling Safe.
        _assertModuleEnabled(callingSafe);

        // Store the configuration for this safe
        _livenessSafeConfiguration[callingSafe] = _config;

        // Clear any existing challenge when configuring/re-configuring.
        // This is necessary because changing the configuration (especially
        // livenessResponsePeriod)
        // would invalidate any ongoing challenge timing, creating inconsistent state.
        // For example, if a challenge was started with a 7-day period and we reconfigure to
        // 1 day, the challenge timing becomes ambiguous. Canceling ensures clean state.
        // Additionally, a Safe that is able to successfully trigger the configuration function
        // is necessarily live, so cancelling the challenge also makes sense from a
        // theoretical standpoint.
        _cancelChallenge(callingSafe);

        emit ModuleConfigured(msg.sender, _config.livenessResponsePeriod, _config.fallbackOwner);

        // Verify that any other extensions which are enabled on the Safe are configured correctly.
        _checkCombinedConfig(callingSafe);
    }

    /// @notice Clears the module configuration for a Safe.
    /// @dev Note: Clearing the configuration also cancels any ongoing challenges.
    ///      This function is intended for use when a Safe wants to permanently remove
    ///      the LivenessModule2 configuration. Typical usage pattern:
    ///      1. Safe disables the module via ModuleManager.disableModule().
    ///      2. Safe calls this clearLivenessModule() function to remove stored configuration.
    ///      3. If Safe later re-enables the module, it must call configureLivenessModule() again.
    ///      Never calling clearLivenessModule() after disabling keeps configuration data
    ///      persistent for potential future re-enabling.
    function clearLivenessModule() external {
        Safe callingSafe = Safe(payable(msg.sender));

        // Check if the calling safe has configuration set
        _assertModuleConfigured(callingSafe);

        // Check that this module is NOT enabled on the calling Safe
        // This prevents clearing configuration while module is still enabled
        _assertModuleNotEnabled(callingSafe);

        // Erase the configuration data for this safe
        delete _livenessSafeConfiguration[callingSafe];

        // Also clear any active challenge
        _cancelChallenge(callingSafe);
        emit ModuleCleared(address(callingSafe));
    }

    /// @notice Challenges an enabled safe.
    /// @param _safe The Safe address to challenge.
    function challenge(Safe _safe) external {
        // Check if the calling safe has configuration set
        _assertModuleConfigured(_safe);

        // Check that the module is still enabled on the target Safe.
        _assertModuleEnabled(_safe);

        // Check that the caller is the fallback owner
        if (msg.sender != _livenessSafeConfiguration[_safe].fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        // Check that no challenge already exists
        if (challengeStartTime[_safe] != 0) {
            revert LivenessModule2_ChallengeAlreadyExists();
        }

        // Set the challenge start time and emit the event
        challengeStartTime[_safe] = block.timestamp;
        emit ChallengeStarted(address(_safe), block.timestamp);
    }

    /// @notice Responds to a challenge for an enabled safe, canceling it.
    function respond() external {
        Safe callingSafe = Safe(payable(msg.sender));

        // Check if the calling safe has configuration set.
        _assertModuleConfigured(callingSafe);

        // Check that this module is enabled on the calling Safe.
        _assertModuleEnabled(callingSafe);

        // Check that a challenge exists
        uint256 startTime = challengeStartTime[callingSafe];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Cancel the challenge without checking if response period has expired
        // This allows the Safe to respond at any time, providing more flexibility
        _cancelChallenge(callingSafe);
    }

    /// @notice With successful challenge, removes all current owners from enabled safe, appoints
    ///         fallback as sole owner, and sets its quorum to 1.
    /// @dev After ownership transfer, the fallback owner becomes the sole owner and is also still
    ///      configured as the fallback owner. If the fallback owner would become unable to sign,
    ///      it would not be able challenge the safe again. For this reason, it is important that
    ///      the fallback owner has a way to preserve its own liveness.
    ///
    ///      It is of critical importance that this function never reverts. If it were to do so,
    ///      the Safe would be permanently bricked. For this reason, the external calls from this
    ///      function are allowed to fail silently instead of reverting.
    /// @param _safe The Safe address to transfer ownership of.
    function changeOwnershipToFallback(Safe _safe) external {
        // Ensure Safe is configured with this module to prevent unauthorized execution.
        _assertModuleConfigured(_safe);

        // Verify module is still enabled to ensure Safe hasn't disabled it mid-challenge.
        _assertModuleEnabled(_safe);

        // Only fallback owner can execute ownership transfer (per specs update)
        if (msg.sender != _livenessSafeConfiguration[_safe].fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        // Verify active challenge exists - without challenge, ownership transfer not allowed
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Ensure response period has fully expired before allowing ownership transfer.
        // This gives Safe owners full configured time to demonstrate liveness.
        if (block.timestamp < getChallengePeriodEnd(_safe)) {
            revert LivenessModule2_ResponsePeriodActive();
        }

        // Reset the challenge state to allow a new challenge
        delete challengeStartTime[_safe];

        // Get current owners
        address[] memory owners = _safe.getOwners();

        // Remove all owners after the first one
        // Note: This loop is safe as real-world Safes have limited owners (typically < 10)
        // Gas limits would only be a concern with hundreds/thousands of owners
        while (owners.length > 1) {
            _safe.execTransactionFromModule({
                to: address(_safe),
                value: 0,
                operation: Enum.Operation.Call,
                data: abi.encodeCall(OwnerManager.removeOwner, (SENTINEL_OWNER, owners[0], 1))
            });
            owners = _safe.getOwners();
        }

        // Now swap the remaining single owner with the fallback owner
        // Note: If the fallback owner would be the only or the last owner in the owners list,
        // swapOwner would internally revert in OwnerManager, but we ignore it because the final
        // owners list would still be what we want.
        _safe.execTransactionFromModule({
            to: address(_safe),
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(
                OwnerManager.swapOwner, (SENTINEL_OWNER, owners[0], _livenessSafeConfiguration[_safe].fallbackOwner)
            )
        });

        // Sanity check: verify the fallback owner is now the only owner
        address[] memory finalOwners = _safe.getOwners();
        if (finalOwners.length != 1 || finalOwners[0] != _livenessSafeConfiguration[_safe].fallbackOwner) {
            revert LivenessModule2_OwnershipTransferFailed();
        }

        // Disable the guard
        // Note that this will remove whichever guard is currently set on the Safe,
        // even if it is not the SaferSafes guard. This is intentional, as it is possible that the
        // guard was the cause of the liveness failure which resulted in the transfer of ownership to
        // the fallback owner.
        // WARNING: Removing the TimelockGuard from a Safe will make all Scheduled and Cancelled
        // transactions at or below the Safe nonce immediately executable by anyone. To avoid this,
        // particularly in an adversarial environment, it is recommended that the fallback owner is
        // also a Safe, and that the call to `changeOwnershipToFallback` is the first transaction
        // in a batch that also includes as many nonce-bumping no-op transactions through the Safe
        // with the TimelockGuard as needed to increase its nonce above that of all Scheduled and
        // Cancelled transactions.
        _safe.execTransactionFromModule({
            to: address(_safe),
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(GuardManager.setGuard, (address(0)))
        });

        emit ChallengeSucceeded(address(_safe), _livenessSafeConfiguration[_safe].fallbackOwner);
    }

    ////////////////////////////////////////////////////////////////
    //                   Internal View Functions                  //
    ////////////////////////////////////////////////////////////////

    /// @notice Internal helper function which can be overriden in a child contract to check if the
    ///         guard's configuration is valid in the context of other extensions that are enabled
    ///         on the Safe.
    function _checkCombinedConfig(Safe _safe) internal view virtual;

    /// @notice Asserts that the module is configured for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleConfigured(Safe _safe) internal view {
        ModuleConfig storage config = _livenessSafeConfiguration[_safe];
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }
    }

    /// @notice Asserts that the module is enabled for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleEnabled(Safe _safe) internal view {
        if (!_safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }
    }

    /// @notice Asserts that the module is not enabled for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleNotEnabled(Safe _safe) internal view {
        if (_safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleStillEnabled();
        }
    }

    ////////////////////////////////////////////////////////////////
    //             Internal State-Changing Functions              //
    ////////////////////////////////////////////////////////////////

    /// @notice Internal function to cancel a challenge and emit the appropriate event.
    /// @param _safe The Safe address for which to cancel the challenge.
    function _cancelChallenge(Safe _safe) internal {
        // Early return if no challenge exists
        if (challengeStartTime[_safe] == 0) return;

        delete challengeStartTime[_safe];
        emit ChallengeCancelled(address(_safe));
    }
}
