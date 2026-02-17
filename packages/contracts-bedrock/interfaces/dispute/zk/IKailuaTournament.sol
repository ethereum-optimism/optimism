// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { Claim, Duration, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IKailuaVerifier } from "interfaces/dispute/zk/IKailuaVerifier.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";

interface IKailuaTournament is IDisputeGame, ISemver {
    /// @notice Denotes the proven status of the game
    enum ProofStatus {
        NONE,
        FAULT,
        VALIDITY
    }

    /// @notice Emitted when a proof is submitted.
    event Proven(bytes32 indexed signature, ProofStatus indexed status);

    function DISPUTE_GAME_FACTORY() external view returns (IDisputeGameFactory);
    function GAME_TYPE() external view returns (GameType);
    function KAILUA_TREASURY() external view returns (IKailuaTournament);
    function KAILUA_VERIFIER() external view returns (IKailuaVerifier);
    function OPTIMISM_PORTAL() external view returns (IOptimismPortal2);
    function OUTPUT_BLOCK_SPAN() external view returns (uint64);
    function PROPOSAL_BLOBS() external view returns (uint64);
    function PROPOSAL_OUTPUT_COUNT() external view returns (uint64);
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_);
    function appendChild() external;
    function blobsHash() external view returns (bytes32 blobsHash_);
    function childCount() external view returns (uint256 count_);
    function children(uint256) external view returns (IKailuaTournament);
    function contenderDuplicates(uint256) external view returns (uint64);
    function contenderIndex() external view returns (uint64);
    function gameIndex() external view returns (uint256);
    function getChallengerDuration(uint64 asOfTimestamp) external view returns (Duration duration_);
    function initialize() external payable;
    function isViableSignature(bytes32 childSignature) external view returns (bool isViableSignature_);
    function minCreationTime() external view returns (Timestamp minCreationTime_);
    function opponentIndex() external view returns (uint64);
    function parentGame() external view returns (IKailuaTournament parentGame_);
    function proofStatus(bytes32) external view returns (ProofStatus);
    function proposalBlobHashes(uint256) external view returns (Hash);
    function proposer() external view returns (address proposer_);
    function proveOutputFault(
        address[2] memory prHs,
        uint64[2] memory co,
        bytes memory encodedSeal,
        bytes32[2] memory ac,
        uint256 proposedOutputFe,
        bytes[][2] memory kzgCommitmentsProofs
    )
        external;
    function proveTrailFault(
        address payoutRecipient,
        uint64[2] memory co,
        uint256 proposedOutputFe,
        bytes memory blobCommitment,
        bytes memory kzgProof
    )
        external;
    function proveValidity(
        address payoutRecipient,
        address l1HeadSource,
        uint64 childIndex,
        bytes memory encodedSeal
    )
        external;
    function provenAt(bytes32) external view returns (Timestamp);
    function prover(bytes32) external view returns (address);
    function pruneChildren(uint256 stepLimit) external returns (IKailuaTournament);
    function signature() external view returns (bytes32 signature_);
    function validChildSignature() external view returns (bytes32);
    function verifyIntermediateOutput(
        uint64 outputNumber,
        uint256 outputFe,
        bytes memory blobCommitment,
        bytes memory kzgProof
    )
        external
        returns (bool success);
    function version() external view returns (string memory);
}
