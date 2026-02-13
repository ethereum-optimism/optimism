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
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDelayedWETH } from "interfaces/dispute/IDelayedWETH.sol";

/// @title IZKDisputeGame
/// @notice Interface for the ZKDisputeGame contract.
interface IZKDisputeGame is IDisputeGame, ISemver {
    enum ProposalStatus {
        Unchallenged,
        Challenged,
        UnchallengedAndValidProofProvided,
        ChallengedAndValidProofProvided,
        Resolved
    }

    struct ClaimData {
        uint32 parentIndex;
        ProposalStatus status;
        address challenger;
        address prover;
        Timestamp deadline;
        Claim claim;
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
    function bondDistributionMode() external view returns (BondDistributionMode);
    function disputeGameFactory() external view returns (IDisputeGameFactory);
    function totalBonds() external view returns (uint256);

    function initialize() external payable;
    function l2SequenceNumber() external pure returns (uint256 l2SequenceNumber_);
    function parentIndex() external pure returns (uint32 parentIndex_);
    function absolutePrestate() external pure returns (bytes32 absolutePrestate_);
    function startingBlockNumber() external view returns (uint256 startingBlockNumber_);
    function startingRootHash() external view returns (Hash startingRootHash_);
    function challenge() external payable returns (ProposalStatus);
    function prove(bytes calldata _proofBytes) external returns (ProposalStatus);
    function resolve() external returns (GameStatus);
    function claimCredit(address _recipient) external;
    function closeGame() external;
    function gameOver() external view returns (bool gameOver_);
    function gameType() external pure returns (GameType gameType_);
    function gameCreator() external pure returns (address creator_);
    function rootClaim() external pure returns (Claim rootClaim_);
    function rootClaimByChainId(uint256) external pure returns (Claim rootClaim_);
    function l1Head() external pure returns (Hash l1Head_);
    function extraData() external pure returns (bytes memory extraData_);
    function gameData() external pure returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_);
    function credit(address _recipient) external view returns (uint256 credit_);
    function maxChallengeDuration() external pure returns (Duration maxChallengeDuration_);
    function maxProveDuration() external pure returns (Duration maxProveDuration_);
    function verifier() external pure returns (IZKVerifier verifier_);
    function challengerBond() external pure returns (uint256 challengerBond_);
    function anchorStateRegistry() external pure returns (IAnchorStateRegistry registry_);
    function weth() external pure returns (IDelayedWETH weth_);
    function l2ChainId() external pure returns (uint256 l2ChainId_);
}
