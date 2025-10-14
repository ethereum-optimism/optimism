// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Safe
import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";

/// @title LivenessModule2
/// @notice This module allows challenge-based ownership transfer to a fallback owner
///         when the Safe becomes unresponsive. The fallback owner can initiate a challenge,
///         and if the Safe doesn't respond within the challenge period, ownership transfers
///         to the fallback owner.
/// @dev This is a singleton contract. To use it:
///      1. The Safe must first enable this module using ModuleManager.enableModule()
///      2. The Safe must then configure the module by calling configure() with params
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
    mapping(address => ModuleConfig) public livenessSafeConfiguration;

    /// @notice Mapping from Safe address to active challenge start time (0 if none).
    mapping(address => uint256) public challengeStartTime;

    /// @notice Reserved address used as previous owner to the first owner in a Safe.
    address internal constant SENTINEL_OWNER = address(0x1);

    /// @notice Error for when module is not enabled for the Safe.
    error LivenessModule2_ModuleNotEnabled();

    /// @notice Error for when Safe is not configured for this module.
    error LivenessModule2_ModuleNotConfigured();

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

    /// @notice Returns challenge_start_time + liveness_response_period if challenge exists, or
    ///         0 if not.
    /// @param _safe The Safe address to query.
    /// @return The challenge end timestamp, or 0 if no challenge.
    function getChallengePeriodEnd(address _safe) public view returns (uint256) {
        uint256 startTime = challengeStartTime[_safe];
        if (startTime == 0) {
            return 0;
        }
        ModuleConfig storage config = livenessSafeConfiguration[_safe];
        return startTime + config.livenessResponsePeriod;
    }

    /// @notice Configures the module for a Safe that has already enabled it.
    /// @param _config The configuration parameters for the module containing the response
    ///                period and fallback owner.
    function configureLivenessModule(ModuleConfig memory _config) external {
        // Validate configuration parameters to ensure module can function properly.
        // livenessResponsePeriod must be > 0 to allow time for Safe owners to respond.
        if (_config.livenessResponsePeriod == 0) {
            revert LivenessModule2_InvalidResponsePeriod();
        }
        // fallbackOwner must not be zero address to have a valid ownership recipient.
        if (_config.fallbackOwner == address(0)) {
            revert LivenessModule2_InvalidFallbackOwner();
        }

        // Check that this module is enabled on the calling Safe.
        _assertModuleEnabled(msg.sender);

        // Store the configuration for this safe
        livenessSafeConfiguration[msg.sender] = _config;

        // Clear any existing challenge when configuring/re-configuring.
        // This is necessary because changing the configuration (especially
        // livenessResponsePeriod)
        // would invalidate any ongoing challenge timing, creating inconsistent state.
        // For example, if a challenge was started with a 7-day period and we reconfigure to
        // 1 day, the challenge timing becomes ambiguous. Canceling ensures clean state.
        // Additionally, a Safe that is able to successfully trigger the configuration function
        // is necessarily live, so cancelling the challenge also makes sense from a
        // theoretical standpoint.
        _cancelChallenge(msg.sender);

        emit ModuleConfigured(msg.sender, _config.livenessResponsePeriod, _config.fallbackOwner);

        // Verify that any other extensions which are enabled on the Safe are configured correctly.
        _checkCombinedConfig(Safe(payable(msg.sender)));
    }

    /// @notice Internal helper function which can be overriden in a child contract to check if the guard's
    ///         configuration is valid in the context of other extensions that are enabled on the Safe.
    function _checkCombinedConfig(Safe _safe) internal view virtual;

    /// @notice Clears the module configuration for a Safe.
    /// @dev Note: Clearing the configuration also cancels any ongoing challenges.
    ///      This function is intended for use when a Safe wants to permanently remove
    ///      the LivenessModule2 configuration. Typical usage pattern:
    ///      1. Safe disables the module via ModuleManager.disableModule().
    ///      2. Safe calls this clearLivenessModule() function to remove stored configuration.
    ///      3. If Safe later re-enables the module, it must call configureLivenessModule() again.
    ///      Never calling clearLivenessModule() after disabling keeps configuration data persistent
    ///      for potential future re-enabling.
    function clearLivenessModule() external {
        // Check if the calling safe has configuration set
        _assertModuleConfigured(msg.sender);

        // Check that this module is NOT enabled on the calling Safe
        // This prevents clearing configuration while module is still enabled
        _assertModuleNotEnabled(msg.sender);

        // Erase the configuration data for this safe
        delete livenessSafeConfiguration[msg.sender];
        // Also clear any active challenge
        _cancelChallenge(msg.sender);
        emit ModuleCleared(msg.sender);
    }

    /// @notice Challenges an enabled safe.
    /// @param _safe The Safe address to challenge.
    function challenge(address _safe) external {
        // Check if the calling safe has configuration set
        _assertModuleConfigured(_safe);

        // Check that the module is still enabled on the target Safe.
        _assertModuleEnabled(_safe);

        // Check that the caller is the fallback owner
        if (msg.sender != livenessSafeConfiguration[_safe].fallbackOwner) {
            revert LivenessModule2_UnauthorizedCaller();
        }

        // Check that no challenge already exists
        if (challengeStartTime[_safe] != 0) {
            revert LivenessModule2_ChallengeAlreadyExists();
        }

        // Set the challenge start time and emit the event
        challengeStartTime[_safe] = block.timestamp;
        emit ChallengeStarted(_safe, block.timestamp);
    }

    /// @notice Responds to a challenge for an enabled safe, canceling it.
    function respond() external {
        // Check if the calling safe has configuration set.
        _assertModuleConfigured(msg.sender);

        // Check that this module is enabled on the calling Safe.
        _assertModuleEnabled(msg.sender);

        // Check that a challenge exists
        uint256 startTime = challengeStartTime[msg.sender];
        if (startTime == 0) {
            revert LivenessModule2_ChallengeDoesNotExist();
        }

        // Cancel the challenge without checking if response period has expired
        // This allows the Safe to respond at any time, providing more flexibility
        _cancelChallenge(msg.sender);
    }

    /// @notice With successful challenge, removes all current owners from enabled safe,
    ///         appoints fallback as sole owner, and sets its quorum to 1.
    /// @dev Note: After ownership transfer, the fallback owner becomes the sole owner
    ///      and is also still configured as the fallback owner. This means the
    ///      fallback owner effectively becomes its own fallback owner, maintaining
    ///      the ability to challenge itself if needed.
    /// @param _safe The Safe address to transfer ownership of.
    function changeOwnershipToFallback(address _safe) external {
        // Ensure Safe is configured with this module to prevent unauthorized execution.
        _assertModuleConfigured(_safe);

        // Verify module is still enabled to ensure Safe hasn't disabled it mid-challenge.
        _assertModuleEnabled(_safe);

        // Only fallback owner can execute ownership transfer (per specs update)
        if (msg.sender != livenessSafeConfiguration[_safe].fallbackOwner) {
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

        Safe targetSafe = Safe(payable(_safe));

        // Get current owners
        address[] memory owners = targetSafe.getOwners();

        // Remove all owners after the first one
        // Note: This loop is safe as real-world Safes have limited owners (typically < 10)
        // Gas limits would only be a concern with hundreds/thousands of owners
        while (owners.length > 1) {
            targetSafe.execTransactionFromModule({
                to: _safe,
                value: 0,
                operation: Enum.Operation.Call,
                data: abi.encodeCall(OwnerManager.removeOwner, (SENTINEL_OWNER, owners[0], 1))
            });
            owners = targetSafe.getOwners();
        }

        // Now swap the remaining single owner with the fallback owner
        targetSafe.execTransactionFromModule({
            to: _safe,
            value: 0,
            operation: Enum.Operation.Call,
            data: abi.encodeCall(
                OwnerManager.swapOwner, (SENTINEL_OWNER, owners[0], livenessSafeConfiguration[_safe].fallbackOwner)
            )
        });

        // Sanity check: verify the fallback owner is now the only owner
        address[] memory finalOwners = targetSafe.getOwners();
        if (finalOwners.length != 1 || finalOwners[0] != livenessSafeConfiguration[_safe].fallbackOwner) {
            revert LivenessModule2_OwnershipTransferFailed();
        }

        // Reset the challenge state to allow a new challenge
        delete challengeStartTime[_safe];

        emit ChallengeSucceeded(_safe, livenessSafeConfiguration[_safe].fallbackOwner);
    }

    /// @notice Asserts that the module is configured for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleConfigured(address _safe) internal view {
        ModuleConfig storage config = livenessSafeConfiguration[_safe];
        if (config.fallbackOwner == address(0)) {
            revert LivenessModule2_ModuleNotConfigured();
        }
    }

    /// @notice Asserts that the module is enabled for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleEnabled(address _safe) internal view {
        Safe safe = Safe(payable(_safe));
        if (!safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleNotEnabled();
        }
    }

    /// @notice Asserts that the module is not enabled for the given Safe.
    /// @param _safe The Safe address to check.
    function _assertModuleNotEnabled(address _safe) internal view {
        Safe safe = Safe(payable(_safe));
        if (safe.isModuleEnabled(address(this))) {
            revert LivenessModule2_ModuleStillEnabled();
        }
    }

    /// @notice Internal function to cancel a challenge and emit the appropriate event.
    /// @param _safe The Safe address for which to cancel the challenge.
    function _cancelChallenge(address _safe) internal {
        // Early return if no challenge exists
        if (challengeStartTime[_safe] == 0) return;

        delete challengeStartTime[_safe];
        emit ChallengeCancelled(_safe);
    }
}
