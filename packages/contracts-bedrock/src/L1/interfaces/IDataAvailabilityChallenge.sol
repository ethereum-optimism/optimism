// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IDataAvailabilityChallenge
/// @notice Interface for the DataAvailabilityChallenge contract.
interface IDataAvailabilityChallenge {
    receive() external payable;

    function balances(address) external view returns (uint256);
    function bondSize() external view returns (uint256);
    function challenge(uint256 challengedBlockNumber, bytes memory challengedCommitment) external payable;
    function challengeWindow() external view returns (uint256);
    function deposit() external payable;
    function fixedResolutionCost() external view returns (uint256);
    function initialize(
        address _owner,
        uint256 _challengeWindow,
        uint256 _resolveWindow,
        uint256 _bondSize,
        uint256 _resolverRefundPercentage
    )
        external;
    function resolve(
        uint256 challengedBlockNumber,
        bytes memory challengedCommitment,
        bytes memory resolveData
    )
        external;
    function resolveWindow() external view returns (uint256);
    function resolverRefundPercentage() external view returns (uint256);
    function setBondSize(uint256 _bondSize) external;
    function setResolverRefundPercentage(uint256 _resolverRefundPercentage) external;
    function unlockBond(uint256 challengedBlockNumber, bytes memory challengedCommitment) external;
    function validateCommitment(bytes memory commitment) external pure;
    function variableResolutionCost() external view returns (uint256);
    function variableResolutionCostPrecision() external view returns (uint256);
    function withdraw() external;
}
