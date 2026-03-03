// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import {
    BondDistributionMode,
    Claim,
    Duration,
    GameStatus,
    GameType,
    Hash,
    Timestamp,
    Proposal
} from "src/dispute/lib/Types.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { ISP1Verifier } from "src/dispute/zk/ISP1Verifier.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { AccessManager } from "src/dispute/zk/AccessManager.sol";

/// @title IOptimisticZkGame
/// @notice Interface for the OptimisticZkGame contract.
interface IOptimisticZkGame is IDisputeGame, ISemver {
    enum ProposalStatus {
        Unchallenged,
        Challenged,
        UnchallengedAndValidProofProvided,
        ChallengedAndValidProofProvided,
        Resolved
    }

    struct ClaimData {
        uint32 parentIndex;
        address counteredBy;
        address prover;
        Claim claim;
        ProposalStatus status;
        Timestamp deadline;
    }

    /// @notice Emitted when the game is challenged.
    event Challenged(address indexed challenger);

    /// @notice Emitted when the game is proved.
    event Proved(address indexed prover);

    /// @notice Emitted when the game is closed.
    event GameClosed(BondDistributionMode bondDistributionMode);

    function version() external view returns (string memory);
    function createdAt() external view returns (Timestamp);
    function resolvedAt() external view returns (Timestamp);
    function status() external view returns (GameStatus);
    function claimData() external view returns (ClaimData memory);
    function normalModeCredit(address) external view returns (uint256);
    function refundModeCredit(address) external view returns (uint256);
    function startingProposal() external view returns (Proposal memory);
    function wasRespectedGameTypeWhenCreated() external view returns (bool);
    function bondDistributionMode() external view returns (BondDistributionMode);

    function initialize() external payable;
    function l2SequenceNumber() external pure returns (uint256 l2SequenceNumber_);
    function parentIndex() external pure returns (uint32 parentIndex_);
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_);
    function startingRootHash() external view returns (Hash startingRootHash_);
    function challenge() external payable returns (ProposalStatus);
    function prove(bytes calldata _proofBytes) external returns (ProposalStatus);
    function resolve() external returns (GameStatus);
    function claimCredit(address _recipient) external;
    function closeGame() external;
    function gameOver() external view returns (bool gameOver_);
    function gameType() external view returns (GameType gameType_);
    function gameCreator() external pure returns (address creator_);
    function rootClaim() external pure returns (Claim rootClaim_);
    function rootClaimByChainId(uint256) external pure returns (Claim rootClaim_);
    function l1Head() external pure returns (Hash l1Head_);
    function extraData() external pure returns (bytes memory extraData_);
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_);
    function credit(address _recipient) external view returns (uint256 credit_);
    function maxChallengeDuration() external view returns (Duration maxChallengeDuration_);
    function maxProveDuration() external view returns (Duration maxProveDuration_);
    function disputeGameFactory() external view returns (IDisputeGameFactory disputeGameFactory_);
    function sp1Verifier() external view returns (ISP1Verifier verifier_);
    function rollupConfigHash() external view returns (bytes32 rollupConfigHash_);
    function aggregationVkey() external view returns (bytes32 aggregationVkey_);
    function rangeVkeyCommitment() external view returns (bytes32 rangeVkeyCommitment_);
    function challengerBond() external view returns (uint256 challengerBond_);
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_);
    function accessManager() external view returns (AccessManager accessManager_);
}
