// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGame } from "./IDisputeGame.sol";

import "src/dispute/lib/Types.sol";

/// @title IFaultDisputeGame
/// @notice The interface for a fault proof backed dispute game.
interface IFaultDisputeGame is IDisputeGame {
    /// @notice The `ClaimData` struct represents the data associated with a Claim.
    struct ClaimData {
        uint32 parentIndex;
        address counteredBy;
        address claimant;
        uint128 bond;
        Claim claim;
        Position position;
        Clock clock;
    }

    /// @notice The `ResolutionCheckpoint` struct represents the data associated with an in-progress claim resolution.
    struct ResolutionCheckpoint {
        bool initialCheckpointComplete;
        uint32 subgameIndex;
        Position leftmostPosition;
        address counteredBy;
    }

    /// @notice Emitted when a new claim is added to the DAG by `claimant`
    /// @param parentIndex The index within the `claimData` array of the parent claim
    /// @param claim The claim being added
    /// @param claimant The address of the claimant
    event Move(uint256 indexed parentIndex, Claim indexed claim, address indexed claimant);

    /// @notice Attack a disagreed upon `Claim`.
    /// @param _disputed The `Claim` being attacked.
    /// @param _parentIndex Index of the `Claim` to attack in the `claimData` array. This must match the `_disputed`
    /// claim.
    /// @param _claim The `Claim` at the relative attack position.
    function attack(Claim _disputed, uint256 _parentIndex, Claim _claim) external payable;

    /// @notice Defend an agreed upon `Claim`.
    /// @notice _disputed The `Claim` being defended.
    /// @param _parentIndex Index of the claim to defend in the `claimData` array. This must match the `_disputed`
    /// claim.
    /// @param _claim The `Claim` at the relative defense position.
    function defend(Claim _disputed, uint256 _parentIndex, Claim _claim) external payable;

    /// @notice Perform an instruction step via an on-chain fault proof processor.
    /// @dev This function should point to a fault proof processor in order to execute
    ///      a step in the fault proof program on-chain. The interface of the fault proof
    ///      processor contract should adhere to the `IBigStepper` interface.
    /// @param _claimIndex The index of the challenged claim within `claimData`.
    /// @param _isAttack Whether or not the step is an attack or a defense.
    /// @param _stateData The stateData of the step is the preimage of the claim at the given
    ///        prestate, which is at `_stateIndex` if the move is an attack and `_claimIndex` if
    ///        the move is a defense. If the step is an attack on the first instruction, it is
    ///        the absolute prestate of the fault proof VM.
    /// @param _proof Proof to access memory nodes in the VM's merkle state tree.
    function step(uint256 _claimIndex, bool _isAttack, bytes calldata _stateData, bytes calldata _proof) external;

    /// @notice Posts the requested local data to the VM's `PreimageOralce`.
    /// @param _ident The local identifier of the data to post.
    /// @param _execLeafIdx The index of the leaf claim in an execution subgame that requires the local data for a step.
    /// @param _partOffset The offset of the data to post.
    function addLocalData(uint256 _ident, uint256 _execLeafIdx, uint256 _partOffset) external;

    /// @notice Resolves the subgame rooted at the given claim index. `_numToResolve` specifies how many children of
    ///         the subgame will be checked in this call. If `_numToResolve` is less than the number of children, an
    ///         internal cursor will be updated and this function may be called again to complete resolution of the
    ///         subgame.
    /// @dev This function must be called bottom-up in the DAG
    ///      A subgame is a tree of claims that has a maximum depth of 1.
    ///      A subgame root claims is valid if, and only if, all of its child claims are invalid.
    ///      At the deepest level in the DAG, a claim is invalid if there's a successful step against it.
    /// @param _claimIndex The index of the subgame root claim to resolve.
    /// @param _numToResolve The number of subgames to resolve in this call. If the input is `0`, and this is the first
    ///                      page, this function will attempt to check all of the subgame's children at once.
    function resolveClaim(uint256 _claimIndex, uint256 _numToResolve) external;

    /// @notice Returns the number of children that still need to be resolved in order to fully resolve a subgame rooted
    ///         at `_claimIndex`.
    /// @param _claimIndex The subgame root claim's index within `claimData`.
    /// @return numRemainingChildren_ The number of children that still need to be checked to resolve the subgame.
    function getNumToResolve(uint256 _claimIndex) external view returns (uint256 numRemainingChildren_);

    /// @notice The l2BlockNumber of the disputed output root in the `L2OutputOracle`.
    function l2BlockNumber() external view returns (uint256 l2BlockNumber_);

    /// @notice Starting output root and block number of the game.
    function startingOutputRoot() external view returns (Hash startingRoot_, uint256 l2BlockNumber_);

    /// @notice Only the starting block number of the game.
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_);

    /// @notice Only the starting output root of the game.
    function startingRootHash() external view returns (Hash startingRootHash_);
}
