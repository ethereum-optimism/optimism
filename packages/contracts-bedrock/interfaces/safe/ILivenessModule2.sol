// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ILivenessModule2
/// @notice Interface for LivenessModule2, a singleton module for challenge-based ownership transfer
interface ILivenessModule2 {
    /// @notice Configuration for a Safe's liveness module
    struct ModuleConfig {
        uint256 livenessResponsePeriod;
        address fallbackOwner;
    }

    /// @notice Returns the configuration for a Safe
    /// @return livenessResponsePeriod The response period
    /// @return fallbackOwner The fallback owner address
    function livenessSafeConfiguration(address) external view returns (uint256 livenessResponsePeriod, address fallbackOwner);

    /// @notice Returns the challenge start time for a Safe (0 if no challenge)
    /// @return The challenge start timestamp
    function challengeStartTime(address) external view returns (uint256);

    /// @notice Semantic version
    /// @return version The contract version
    function version() external view returns (string memory);

    /// @notice Configures the module for a Safe that has already enabled it
    /// @param _config The configuration parameters for the module
    function configureLivenessModule(ModuleConfig memory _config) external;

    /// @notice Clears the module configuration for a Safe
    function clearLivenessModule() external;

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
