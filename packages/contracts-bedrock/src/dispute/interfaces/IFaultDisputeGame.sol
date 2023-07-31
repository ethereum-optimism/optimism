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
    /// @param claim The claim being added
    /// @param claimant The address of the claimant
    event Move(uint256 indexed parentIndex, Claim indexed claim, address indexed claimant);

    /// @notice Attack a disagreed upon `Claim`.
    /// @param _parentIndex Index of the `Claim` to attack in `claimData`.
    /// @param _claim The `Claim` at the relative attack position.
    function attack(uint256 _parentIndex, Claim _claim) external payable;

    /// @notice Defend an agreed upon `Claim`.
    /// @param _parentIndex Index of the claim to defend in `claimData`.
    /// @param _claim The `Claim` at the relative defense position.
    function defend(uint256 _parentIndex, Claim _claim) external payable;

    /// @notice Perform the final step via an on-chain fault proof processor
    /// @dev This function should point to a fault proof processor in order to execute
    ///      a step in the fault proof program on-chain. The interface of the fault proof
    ///      processor contract should be generic enough such that we can use different
    ///      fault proof VMs (MIPS, RiscV5, etc.)
    /// @param _claimIndex The index of the challenged claim within `claimData`.
    /// @param _isAttack Whether or not the step is an attack or a defense.
    /// @param _stateData The stateData of the step is the preimage of the claim at the given
    ///        prestate, which is at `_stateIndex` if the move is an attack and `_claimIndex` if
    ///        the move is a defense. If the step is an attack on the first instruction, it is
    ///        the absolute prestate of the fault proof VM.
    /// @param _proof Proof to access memory leaf nodes in the VM.
    function step(
        uint256 _claimIndex,
        bool _isAttack,
        bytes calldata _stateData,
        bytes calldata _proof
    ) external;

    /// @notice Posts the requested local data to the VM's `PreimageOralce`.
    /// @param _ident The local identifier of the data to post.
    /// @param _partOffset The offset of the data to post.
    function addLocalData(uint256 _ident, uint256 _partOffset) external;

    /// @notice Returns the L1 block hash at the time of the game's creation.
    function l1Head() external view returns (Hash l1Head_);

    /// @notice The l2BlockNumber that the `rootClaim` commits to. The trace being bisected within
    ///         the game is from `l2BlockNumber - 1` -> `l2BlockNumber`.
    /// @return l2BlockNumber_ The l2BlockNumber that the `rootClaim` commits to.
    function l2BlockNumber() external view returns (uint256 l2BlockNumber_);
}
