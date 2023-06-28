// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "../../libraries/DisputeTypes.sol";

import { IDisputeGame } from "./IDisputeGame.sol";

/// @title IFaultDisputeGame
/// @notice The interface for a fault proof backed dispute game.
interface IFaultDisputeGame is IDisputeGame {
    /// @notice The `ClaimData` struct represents the data associated with a Claim.
    /// @dev TODO: Add bond ID information.
    struct ClaimData {
        uint32 parentIndex;
        bool countered;
        Claim claim;
        Position position;
        Clock clock;
    }

    /// @notice Emitted when a new claim is added to the DAG by `claimant`
    /// @param parentIndex The index within the `claimData` array of the parent claim
    /// @param pivot The claim being added
    /// @param claimant The address of the claimant
    event Move(uint256 indexed parentIndex, Claim indexed pivot, address indexed claimant);

    /// @notice Attack a disagreed upon `Claim`.
    /// @param _parentIndex Index of the `Claim` to attack in `claimData`.
    /// @param _pivot The `Claim` at the relative attack position.
    function attack(uint256 _parentIndex, Claim _pivot) external payable;

    /// @notice Defend an agreed upon `Claim`.
    /// @param _parentIndex Index of the claim to defend in `claimData`.
    /// @param _pivot The `Claim` at the relative defense position.
    function defend(uint256 _parentIndex, Claim _pivot) external payable;

    /// @notice Perform the final step via an on-chain fault proof processor
    /// @dev This function should point to a fault proof processor in order to execute
    ///      a step in the fault proof program on-chain. The interface of the fault proof
    ///      processor contract should be generic enough such that we can use different
    ///      fault proof VMs (MIPS, RiscV5, etc.)
    /// @param _stateIndex The index of the pre/post state of the step within `claimData`.
    /// @param _claimIndex The index of the challenged claim within `claimData`.
    /// @param _isAttack Whether or not the step is an attack or a defense.
    /// @param _stateData The stateData of the step is the preimage of the claim @ `prestateIndex`
    /// @param _proof Proof to access memory leaf nodes in the VM.
    function step(
        uint256 _stateIndex,
        uint256 _claimIndex,
        bool _isAttack,
        bytes calldata _stateData,
        bytes calldata _proof
    ) external;

    /// @notice The l2BlockNumber that the `rootClaim` commits to. The trace being bisected within
    ///         the game is from `l2BlockNumber - 1` -> `l2BlockNumber`.
    /// @return l2BlockNumber_ The l2BlockNumber that the `rootClaim` commits to.
    function l2BlockNumber() external view returns (uint256 l2BlockNumber_);
}
