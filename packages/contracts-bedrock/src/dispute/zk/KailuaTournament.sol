// Copyright 2024, 2025 RISC Zero, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// SPDX-License-Identifier: Apache-2.0
pragma solidity 0.8.24;

import { IAnchorStateRegistry } from "interfaces/dispute/IAnchorStateRegistry.sol";
import { IDisputeGame } from "interfaces/dispute/IDisputeGame.sol";
import { IDisputeGameFactory } from "interfaces/dispute/IDisputeGameFactory.sol";
import { IKailuaTournament } from "interfaces/dispute/zk/IKailuaTournament.sol";
import { IKailuaTreasury } from "interfaces/dispute/zk/IKailuaTreasury.sol";
import { IKailuaVerifier } from "interfaces/dispute/zk/IKailuaVerifier.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { Clone } from "@solady/utils/Clone.sol";
import { Claim, Duration, GameStatus, GameType, Hash, Timestamp } from "src/dispute/lib/Types.sol";
import { KailuaLib } from "src/dispute/zk/KailuaLib.sol";
import {
    AlreadyInitialized,
    Blacklisted,
    UnknownGame,
    InvalidParent,
    ClaimAlreadyResolved,
    NotProposed,
    GameNotInProgress,
    InvalidDisputedClaimIndex,
    NoConflict,
    UnknownGame,
    AlreadyProven,
    GameNotResolved,
    NotProven
} from "src/dispute/lib/Errors.sol";

abstract contract KailuaTournament is Clone, IDisputeGame, ISemver {
    /// @notice Semantic version.
    /// @custom:semver 1.2.0
    string public constant version = "1.2.0";

    /// @notice Denotes the proven status of the game
    enum ProofStatus {
        NONE,
        FAULT,
        VALIDITY
    }

    /// @notice Emitted when a proof is submitted.
    event Proven(bytes32 indexed signature, ProofStatus indexed status);

    // ------------------------------
    // Immutable configuration
    // ------------------------------

    /// @notice The Kailua Treasury Implementation contract address
    IKailuaTreasury public immutable KAILUA_TREASURY;

    /// @notice The Kailua Verifier contract
    IKailuaVerifier public immutable KAILUA_VERIFIER;

    /// @notice The number of outputs a proposal must publish
    uint64 public immutable PROPOSAL_OUTPUT_COUNT;

    /// @notice The number of blocks each output must cover
    uint64 public immutable OUTPUT_BLOCK_SPAN;

    /// @notice The number of blobs a claim must provide
    uint64 public immutable PROPOSAL_BLOBS;

    /// @notice The game type ID
    GameType public immutable GAME_TYPE;

    /// @notice The OptimismPortal2 instance
    IOptimismPortal2 public immutable OPTIMISM_PORTAL;

    /// @notice The DisputeGameFactory instance
    IDisputeGameFactory public immutable DISPUTE_GAME_FACTORY;

    constructor(
        IKailuaTreasury _kailuaTreasury,
        IKailuaVerifier _kailuaVerifier,
        uint64 _proposalOutputCount,
        uint64 _outputBlockSpan,
        GameType _gameType,
        IOptimismPortal2 _optimismPortal
    ) {
        KAILUA_TREASURY = _kailuaTreasury;
        KAILUA_VERIFIER = _kailuaVerifier;
        PROPOSAL_OUTPUT_COUNT = _proposalOutputCount;
        OUTPUT_BLOCK_SPAN = _outputBlockSpan;
        // discard published root commitment in calldata
        _proposalOutputCount--;
        PROPOSAL_BLOBS = (_proposalOutputCount / uint64(KailuaLib.FIELD_ELEMENTS_PER_BLOB))
            + ((_proposalOutputCount % uint64(KailuaLib.FIELD_ELEMENTS_PER_BLOB)) == 0 ? 0 : 1);
        GAME_TYPE = _gameType;
        OPTIMISM_PORTAL = _optimismPortal;
        DISPUTE_GAME_FACTORY = OPTIMISM_PORTAL.disputeGameFactory();
    }

    function initializeInternal() internal {
        // INVARIANT: The game must not have already been initialized.
        if (createdAt.raw() > 0) revert AlreadyInitialized();

        // Allow only the treasury to create new games
        if (gameCreator() != address(KAILUA_TREASURY)) {
            revert Blacklisted(gameCreator(), address(KAILUA_TREASURY));
        }

        // Set the game's starting timestamp
        createdAt = Timestamp.wrap(uint64(block.timestamp));

        // Set the game's index in the factory
        gameIndex = DISPUTE_GAME_FACTORY.gameCount();

        // Read respected status
        wasRespectedGameTypeWhenCreated = OPTIMISM_PORTAL.respectedGameType().raw() == GAME_TYPE.raw();
    }

    // ------------------------------
    // Game State
    // ------------------------------

    /// @notice The blob hashes used to create the game
    Hash[] public proposalBlobHashes;

    /// @notice The game's index in the factory
    uint256 public gameIndex;

    /// @notice The address of the prover of a proposal signature
    mapping(bytes32 => address) public prover;

    /// @notice The timestamp of when the first proof for a proposal signature was made
    mapping(bytes32 => Timestamp) public provenAt;

    /// @notice The current proof status of a proposal signature
    mapping(bytes32 => ProofStatus) public proofStatus;

    /// @notice The proposals extending this proposal
    KailuaTournament[] public children;

    /// @notice The first surviving contender
    uint64 public contenderIndex;

    /// @notice Duplicates of the last surviving contender proposal
    uint64[] public contenderDuplicates;

    /// @notice The next unprocessed opponent
    uint64 public opponentIndex;

    /// @notice The signature of the child accepted through a validity proof
    bytes32 public validChildSignature;

    /// @notice Returns the hash of the output claim and all blob hashes associated with this proposal
    function signature() public view returns (bytes32 signature_) {
        // note: the absence of the l1Head in the signature implies that
        // proofs will eventually demonstrate derivation
        signature_ = sha256(abi.encodePacked(rootClaim().raw(), proposalBlobHashes));
    }

    /// @notice Returns whether a child can be considered valid
    function isViableSignature(bytes32 childSignature) public view returns (bool isViableSignature_) {
        if (validChildSignature != 0) {
            isViableSignature_ = childSignature == validChildSignature;
        } else {
            isViableSignature_ = proofStatus[childSignature] != ProofStatus.FAULT;
        }
    }

    /// @notice Returns the address of the prover of the specified signature or the prover of the valid signature
    function getPayoutRecipient(bytes32 childSignature) internal view returns (address payoutRecipient) {
        // The successful exclusive permit owner receives the payout.
        payoutRecipient = KAILUA_VERIFIER.faultProofPermitBeneficiary(IKailuaTournament(address(this)), childSignature);
        // If none exists, then the successful fault prover is the recipient.
        if (payoutRecipient == address(0x0)) {
            payoutRecipient = prover[childSignature];
        }
        // Otherwise, the successful validity prover receives the payout.
        if (payoutRecipient == address(0x0)) {
            payoutRecipient = prover[validChildSignature];
        }
        // Otherwise the child signature is viable and there is no recipient.
    }

    /// @notice Returns true iff the child proposal was eliminated
    function isChildEliminated(KailuaTournament child) internal view returns (bool) {
        address _proposer = KAILUA_TREASURY.proposerOf(address(child));
        uint256 eliminationRound = KAILUA_TREASURY.eliminationRound(_proposer);
        if (eliminationRound == 0 || eliminationRound > child.gameIndex()) {
            // This proposer has not been eliminated as of their proposal at gameIndex
            return false;
        }
        return true;
    }

    /// @notice Returns the number of children
    function childCount() external view returns (uint256 count_) {
        count_ = children.length;
    }

    /// @notice Registers a new proposal that extends this one
    function appendChild() external {
        // INVARIANT: The calling contract is a newly deployed contract by the dispute game factory
        if (!KAILUA_TREASURY.isProposing()) {
            revert UnknownGame();
        }

        // INVARIANT: The calling KailuaGame contract is not referring to itself as a parent
        if (msg.sender == address(this)) {
            revert InvalidParent();
        }

        // INVARIANT: No longer accept proposals after resolution
        if (contenderIndex < children.length && children[contenderIndex].status() == GameStatus.DEFENDER_WINS) {
            revert ClaimAlreadyResolved();
        }

        // Append new child to children list
        children.push(KailuaTournament(msg.sender));
    }

    /// @notice Returns the amount of time left for challenges as of the input timestamp.
    function getChallengerDuration(uint256 asOfTimestamp) public view virtual returns (Duration duration_);

    /// @notice Returns the earliest time at which this proposal could have been created
    function minCreationTime() public view virtual returns (Timestamp minCreationTime_);

    /// @notice Returns the parent game contract.
    function parentGame() public view virtual returns (KailuaTournament parentGame_);

    /// @notice Returns the proposer address
    function proposer() public view returns (address proposer_) {
        proposer_ = KAILUA_TREASURY.proposerOf(address(this));
    }

    /// @notice Verifies that an intermediate output was part of the proposal
    function verifyIntermediateOutput(
        uint64 outputNumber,
        uint256 outputFe,
        bytes calldata blobCommitment,
        bytes calldata kzgProof
    )
        external
        virtual
        returns (bool success);

    /// @notice Updates the provability of a child signature if not already set
    function updateProofStatus(address payoutRecipient, bytes32 childSignature, ProofStatus outcome) internal {
        // INVARIANT: Proofs can only be submitted once
        if (proofStatus[childSignature] != ProofStatus.NONE) {
            revert AlreadyProven();
        }

        // Update proof status
        proofStatus[childSignature] = outcome;

        // Announce proof status
        emit Proven(childSignature, outcome);

        // Set the game's prover address
        prover[childSignature] = payoutRecipient;

        // Set the game's proving timestamp
        provenAt[childSignature] = Timestamp.wrap(uint64(block.timestamp));
    }

    // ------------------------------
    // IDisputeGame implementation
    // ------------------------------

    /// @inheritdoc IDisputeGame
    Timestamp public createdAt;

    /// @inheritdoc IDisputeGame
    Timestamp public resolvedAt;

    /// @inheritdoc IDisputeGame
    GameStatus public status;

    /// @inheritdoc IDisputeGame
    function gameType() external view returns (GameType gameType_) {
        gameType_ = GAME_TYPE;
    }

    /// @inheritdoc IDisputeGame
    function gameCreator() public pure returns (address creator_) {
        creator_ = _getArgAddress(0x00);
    }

    /// @inheritdoc IDisputeGame
    function rootClaimByChainId(uint256) public pure returns (Claim rootClaim_) {
        // todo: support chain id specific proposals
        rootClaim_ = rootClaim();
    }

    /// @inheritdoc IDisputeGame
    function rootClaim() public pure returns (Claim rootClaim_) {
        rootClaim_ = Claim.wrap(_getArgBytes32(0x14));
    }

    /// @inheritdoc IDisputeGame
    function l1Head() public pure returns (Hash l1Head_) {
        l1Head_ = Hash.wrap(_getArgBytes32(0x34));
    }

    /// @notice The l2SequenceNumber of the claim's output root.
    function l2SequenceNumber() public pure returns (uint256 l2SequenceNumber_) {
        l2SequenceNumber_ = uint256(_getArgUint64(0x54));
    }

    /// @inheritdoc IDisputeGame
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_) {
        gameType_ = GAME_TYPE;
        rootClaim_ = this.rootClaim();
        extraData_ = this.extraData();
    }

    /// @notice True iff the Kailua GameType was respected by OptimismPortal at time of creation
    bool public wasRespectedGameTypeWhenCreated;

    /// @notice This is a workaround for withdrawal compatibility under op-contracts v5.0.0
    function anchorStateRegistry() external view returns (IAnchorStateRegistry registry_) {
        registry_ = IAnchorStateRegistry(msg.sender);
    }

    // ------------------------------
    // Tournament
    // ------------------------------

    /// @notice Eliminates children until at least one remains
    function pruneChildren(uint256 stepLimit) external returns (KailuaTournament) {
        // INVARIANT: Only finalized proposals may prune tournaments
        if (status != GameStatus.DEFENDER_WINS) {
            revert GameNotResolved();
        }

        // INVARIANT: No tournament to play without at least one child
        if (children.length == 0) {
            revert NotProposed();
        }

        // Resume from prior surviving contender
        uint64 u = contenderIndex;
        // Resume from prior unprocessed opponent
        uint64 v = opponentIndex;
        // Abort if out of bounds
        if (u == children.length) {
            return KailuaTournament(address(0x0));
        }
        // Advance v if needed
        if (v <= u) {
            // INVARIANT: contenderDuplicates is empty
            v = u + 1;
        }

        // Note: u < children.length
        // Fetch contender details
        KailuaTournament contender = children[u];
        bytes32 contenderSignature = contender.signature();

        // Ensure survivor decision finality after resolution
        if (contender.status() == GameStatus.DEFENDER_WINS) {
            return contender;
        }

        // If the contender is invalid then we eliminate it and find the next viable contender using the opponent
        // pointer. This search could terminate early if the elimination limit is reached.
        // If the contender is valid and its proposer is not eliminated, this is skipped.
        if (!isViableSignature(contenderSignature) || isChildEliminated(contender)) {
            // INVARIANT: If branch entered through isChildEliminated condition, contenderDuplicates is empty

            // Eliminate duplicates
            address payoutRecipient = getPayoutRecipient(contenderSignature);
            for (uint256 i = contenderDuplicates.length; i > 0 && stepLimit > 0; (i--, stepLimit--)) {
                KailuaTournament duplicate = children[contenderDuplicates[i - 1]];
                if (!isChildEliminated(duplicate)) {
                    KAILUA_TREASURY.eliminate(address(duplicate), payoutRecipient);
                }
                contenderDuplicates.pop();
            }

            // Abort if elimination allowance exhausted before eliminating all duplicate contenders
            if (stepLimit == 0) {
                return KailuaTournament(address(0x0));
            }

            // Eliminate contender
            if (!isChildEliminated(contender)) {
                KAILUA_TREASURY.eliminate(address(contender), payoutRecipient);
            }
            stepLimit--;

            // Find next viable contender
            // INVARIANT: v > max(u, contenderDuplicates);
            u = v;
            for (; u < children.length && stepLimit > 0; (u++, stepLimit--)) {
                // Skip if previously eliminated
                contender = children[u];
                if (isChildEliminated(contender)) {
                    continue;
                }
                // Eliminate if faulty
                contenderSignature = contender.signature();
                if (!isViableSignature(contenderSignature)) {
                    // eliminate the unviable contender
                    KAILUA_TREASURY.eliminate(address(contender), getPayoutRecipient(contenderSignature));
                    continue;
                }
                // Select u as next viable contender
                break;
            }
            // Store contender
            contenderIndex = u;
            // Select the next possible opponent
            v = u + 1;
        }

        // Eliminate faulty opponents if we've landed on a viable contender
        if (u < children.length && isViableSignature(children[u].signature())) {
            // Iterate over opponents to eliminate them
            for (; v < children.length && stepLimit > 0; (v++, stepLimit--)) {
                KailuaTournament opponent = children[v];
                // If the contender hasn't been challenged for as long as the timeout, declare them winner
                if (contender.getChallengerDuration(opponent.createdAt().raw()).raw() == 0) {
                    // Note: This implies eliminationLimit > 0
                    break;
                }
                // If the opponent proposer is eliminated, skip
                if (isChildEliminated(opponent)) {
                    continue;
                }
                // Append contender duplicate
                bytes32 opponentSignature = opponent.signature();
                if (opponentSignature == contenderSignature) {
                    contenderDuplicates.push(v);
                    continue;
                }
                // If there is insufficient proof data, abort
                // Validity: The contender is the proven child, the opponent must be incorrect
                // Fault: The contender is not proven faulty, the opponent may (not) be.
                if (isViableSignature(opponentSignature)) {
                    revert NotProven();
                }
                // eliminate the opponent with the unviable proposal
                KAILUA_TREASURY.eliminate(address(opponent), getPayoutRecipient(opponentSignature));
            }

            // INVARIANT: v > u && contender == children[u]
            // Record incremental opponent elimination progress
            opponentIndex = v;

            // Return the sole survivor if no more matches can be played
            if (v == children.length || stepLimit > 0) {
                return contender;
            }
        }

        // No survivor yet
        return KailuaTournament(address(0x0));
    }

    // ------------------------------
    // Validity proving
    // ------------------------------

    /// @notice Returns the hash of all blob hashes associated with this proposal
    function blobsHash() public view returns (bytes32 blobsHash_) {
        blobsHash_ = sha256(abi.encodePacked(proposalBlobHashes));
    }

    /// @notice Proves that a proposal is valid
    function proveValidity(
        address payoutRecipient,
        address l1HeadSource,
        uint64 childIndex,
        bytes calldata encodedSeal
    )
        external
    {
        KailuaTournament childContract = children[childIndex];
        // INVARIANT: Can only prove validity of unresolved proposals
        if (childContract.status() != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // Store validity proof data (deleted on revert)
        validChildSignature = childContract.signature();

        // INVARIANT: No longer accept proofs after resolution
        if (contenderIndex < children.length && children[contenderIndex].status() == GameStatus.DEFENDER_WINS) {
            revert ClaimAlreadyResolved();
        }

        // Calculate the expected precondition hash if blob data is necessary for proposal
        bytes32 preconditionHash = bytes32(0x0);
        if (PROPOSAL_OUTPUT_COUNT > 1) {
            preconditionHash = sha256(
                abi.encodePacked(
                    uint64(l2SequenceNumber()),
                    uint64(PROPOSAL_OUTPUT_COUNT),
                    uint64(OUTPUT_BLOCK_SPAN),
                    childContract.blobsHash()
                )
            );
        }

        // update proof status
        prove(
            l1HeadSource,
            payoutRecipient,
            preconditionHash,
            rootClaim().raw(),
            childContract.rootClaim().raw(),
            PROPOSAL_OUTPUT_COUNT,
            encodedSeal,
            validChildSignature,
            ProofStatus.VALIDITY
        );
    }

    // ------------------------------
    // Fault proving
    // ------------------------------

    /// @notice Proves that a proposal committed to an incorrect transition
    function proveOutputFault(
        // [ payoutRecipient, l1HeadSource ]
        address[2] calldata prHs,
        // [ childIndex, outputOffset ]
        uint64[2] calldata co,
        bytes calldata encodedSeal,
        // [ acceptedOutputHash, computedOutputHash ]
        bytes32[2] memory ac,
        uint256 proposedOutputFe,
        bytes[][2] calldata kzgCommitmentsProofs
    )
        external
    {
        KailuaTournament childContract = children[co[0]];
        // INVARIANT: Proofs cannot be submitted unless the child is playing.
        if (childContract.status() != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // INVARIANT: No longer accept proofs after resolution
        if (contenderIndex < children.length && children[contenderIndex].status() == GameStatus.DEFENDER_WINS) {
            revert ClaimAlreadyResolved();
        }

        // INVARIANT: Proofs can only pertain to intermediate outputs
        if (co[1] >= PROPOSAL_OUTPUT_COUNT) {
            revert InvalidDisputedClaimIndex();
        }

        // Validate the common output root.
        if (co[1] == 0) {
            // Note: acceptedOutputHash cannot be a reduced fe because the comparison below will fail
            // The safe output is the parent game's output when proving the first output
            require(ac[0] == rootClaim().raw(), "bad acceptedOutput");
        } else {
            // Note: acceptedOutputHash cannot be a reduced fe because the journal would not be provable
            // Prove common output publication
            require(
                childContract.verifyIntermediateOutput(
                    co[1] - 1, KailuaLib.hashToFe(ac[0]), kzgCommitmentsProofs[0][0], kzgCommitmentsProofs[1][0]
                ),
                "bad acceptedOutput kzg"
            );
        }

        // Validate the claimed output root.
        if (co[1] == PROPOSAL_OUTPUT_COUNT - 1) {
            // INVARIANT: Proofs can only show disparities
            if (ac[1] == childContract.rootClaim().raw()) {
                revert NoConflict();
            }
        } else {
            // Note: proposedOutputFe must be a canonical point or point eval precompile call will fail
            // Prove divergent output publication
            require(
                childContract.verifyIntermediateOutput(
                    co[1],
                    proposedOutputFe,
                    kzgCommitmentsProofs[0][kzgCommitmentsProofs[0].length - 1],
                    kzgCommitmentsProofs[1][kzgCommitmentsProofs[1].length - 1]
                ),
                "bad proposedOutput kzg"
            );
            // INVARIANT: Proofs can only show disparities
            if (KailuaLib.hashToFe(ac[1]) == proposedOutputFe) {
                revert NoConflict();
            }
        }

        // update proof status
        prove(
            prHs[1],
            prHs[0],
            bytes32(0),
            ac[0],
            ac[1],
            co[1] + 1,
            encodedSeal,
            childContract.signature(),
            ProofStatus.FAULT
        );
    }

    /// @notice Proves that a proposal contains invalid intermediate data
    function proveTrailFault(
        address payoutRecipient,
        uint64[2] calldata co,
        uint256 proposedOutputFe,
        bytes calldata blobCommitment,
        bytes calldata kzgProof
    )
        external
    {
        KailuaTournament childContract = children[co[0]];
        // INVARIANT: Proofs cannot be submitted unless the children are playing.
        if (childContract.status() != GameStatus.IN_PROGRESS) {
            revert GameNotInProgress();
        }

        // INVARIANT: No longer accept proofs after resolution
        if (contenderIndex < children.length && children[contenderIndex].status() == GameStatus.DEFENDER_WINS) {
            revert ClaimAlreadyResolved();
        }

        // INVARIANT: Proofs can only pertain to trail data
        if (co[1] < PROPOSAL_OUTPUT_COUNT) {
            revert InvalidDisputedClaimIndex();
        }

        // We expect all trail data to be zeroed
        if (proposedOutputFe == 0) {
            revert NoConflict();
        }

        // Because the root claim is considered the last published output, we shift the provided  output offset down by
        // one to correctly point to the target trailing zero output
        // INVARIANT: The divergence occurs in the last blob
        uint64 feOffset = co[1] - 1;
        if (KailuaLib.blobIndex(feOffset) != PROPOSAL_BLOBS - 1) {
            revert InvalidDisputedClaimIndex();
        }

        // Validate the claimed output root publications
        // Note: proposedOutputFe must be a canonical field element or point eval precompile call will fail
        require(
            childContract.verifyIntermediateOutput(feOffset, proposedOutputFe, blobCommitment, kzgProof),
            "bad proposedOutput kzg"
        );

        // Update dispute status based on trailing data
        updateProofStatus(payoutRecipient, childContract.signature(), ProofStatus.FAULT);
    }

    // ------------------------------
    // ZK Proving
    // ------------------------------

    /// @notice Verifies a ZK proof and updates the proof status according to the provided outcome if the proof is valid
    function prove(
        address l1HeadSource,
        address payoutRecipient,
        bytes32 preconditionHash,
        bytes32 acceptedOutputHash,
        bytes32 computedOutputHash,
        uint64 outputCount,
        bytes calldata encodedSeal,
        bytes32 childSignature,
        ProofStatus outcome
    )
        internal
    {
        // Validate the l1Head source
        if (KAILUA_TREASURY.proposerOf(l1HeadSource) == address(0x0)) {
            revert UnknownGame();
        }

        // Revert on proof verification failure
        KAILUA_VERIFIER.verify(
            // The address of the recipient of the payout for this proof
            payoutRecipient,
            // The blob equivalence precondition hash
            preconditionHash,
            // The L1 head hash containing the safe L2 chain data that may reproduce the L2 head hash.
            KailuaTournament(l1HeadSource).l1Head().raw(),
            // The accepted output
            acceptedOutputHash,
            // The proposed output
            computedOutputHash,
            // The claim block number
            uint64(l2SequenceNumber() + outputCount * OUTPUT_BLOCK_SPAN),
            // The cryptographic proof
            encodedSeal
        );

        // Mark the child as proven
        updateProofStatus(payoutRecipient, childSignature, outcome);
    }
}
