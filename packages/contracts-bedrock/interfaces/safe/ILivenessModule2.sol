// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title ILivenessModule2
/// @notice Interface for LivenessModule2, a singleton module for challenge-based ownership transfer
interface ILivenessModule2 is ISemver {

    /// @notice Configuration for a Safe's liveness module
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
    }

    /// @notice Error for when module is not enabled for the Safe
    error LivenessModule2_ModuleNotEnabled();

    /// @notice Error for when Safe is not configured for this module
    error LivenessModule2_ModuleNotConfigured();

    /// @notice Error for when a challenge already exists
    error LivenessModule2_ChallengeAlreadyExists();

    /// @notice Error for when no challenge exists
    error LivenessModule2_ChallengeDoesNotExist();

    /// @notice Error for when trying to cancel a challenge after the response period has ended
    error LivenessModule2_ResponsePeriodEnded();

    /// @notice Error for when trying to execute ownership transfer while the response period is still active
    error LivenessModule2_ResponsePeriodActive();

    /// @notice Error for when caller is not authorized
    error LivenessModule2_UnauthorizedCaller();

    /// @notice Error for invalid parameters
    error LivenessModule2_InvalidParameters();

    /// @notice Error for when trying to clear configuration while module is still enabled
    error LivenessModule2_ModuleStillEnabled();

    /// @notice Emitted when a Safe enables the module
    event ModuleEnabled(address indexed safe, uint256 livenessResponsePeriod, address fallbackOwner);

    /// @notice Emitted when a Safe disables the module
    event ModuleDisabled(address indexed safe);

    /// @notice Emitted when a challenge is started
    event ChallengeStarted(address indexed safe, uint256 challengeStartTime);

    /// @notice Emitted when a challenge is cancelled
    event ChallengeCancelled(address indexed safe);

    /// @notice Emitted when ownership is transferred to the fallback owner
    event ChallengeExecuted(address indexed safe, address fallbackOwner);

    /// @notice Reserved address used as the previous owner to the first owner in a Safe
    /// @return The sentinel owner address (0x1)
    function SENTINEL_OWNER() external view returns (address);

    /// @notice Returns the configuration for a Safe
    /// @return livenessResponsePeriod The response period
    /// @return fallbackOwner The fallback owner address
    function safeConfigs(address) external view returns (uint256 livenessResponsePeriod, address fallbackOwner);

    /// @notice Returns the challenge start time for a Safe (0 if no challenge)
    /// @return The challenge start timestamp
    function challengeStartTime(address) external view returns (uint256);

    /// @notice Semantic version
    /// @return version The contract version
    function version() external view returns (string memory);

    /// @notice Configures the module for a Safe that has already enabled it
    /// @param _config The configuration parameters for the module
    function configure(ModuleConfig memory _config) external;

    /// @notice Clears the module configuration for a Safe
    function clear() external;

    /// @notice Returns challenge_start_time + liveness_response_period if there is a challenge, or 0 if not
    /// @param _safe The Safe address to query
    /// @return The challenge end timestamp, or 0 if no challenge
    function getChallengePeriodEnd(address _safe) external view returns (uint256);

    /// @notice Challenges an enabled safe
    /// @param _safe The Safe to challenge
    function challenge(address _safe) external;

    /// @notice Responds to a challenge for an enabled safe, canceling it
    function respond() external;

    /// @notice Removes all current owners from an enabled safe and appoints fallback as sole owner
    /// @param _safe The Safe to transfer ownership of
    function changeOwnershipToFallback(address _safe) external;
}
