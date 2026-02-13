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

import "interfaces/dispute/zk/IKailuaTreasury.sol";
import "interfaces/dispute/zk/IKailuaVerifier.sol";
import "src/dispute/lib/Types.sol";


interface IKailuaTournament {
    /// @notice Denotes the proven status of the game
    enum ProofStatus {
        NONE,
        FAULT,
        VALIDITY
    }

    /// @notice Emitted when a proof is submitted.
    event Proven(bytes32 indexed signature, ProofStatus indexed status);

    function DISPUTE_GAME_FACTORY() external view returns (address);
    function GAME_TYPE() external view returns (GameType);
    function KAILUA_TREASURY() external view returns (IKailuaTreasury);
    function KAILUA_VERIFIER() external view returns (IKailuaVerifier);
    function OPTIMISM_PORTAL() external view returns (address);
    function OUTPUT_BLOCK_SPAN() external view returns (uint64);
    function PROPOSAL_BLOBS() external view returns (uint64);
    function PROPOSAL_OUTPUT_COUNT() external view returns (uint64);
    function anchorStateRegistry() external view returns (address registry_);
    function appendChild() external;
    function blobsHash() external view returns (bytes32 blobsHash_);
    function childCount() external view returns (uint256 count_);
    function children(uint256) external view returns (address);
    function contenderDuplicates(uint256) external view returns (uint64);
    function contenderIndex() external view returns (uint64);
    function createdAt() external view returns (Timestamp);
    function extraData() external pure returns (bytes memory extraData_);
    function gameCreator() external pure returns (address creator_);
    function gameData() external view returns (GameType gameType_, Claim rootClaim_, bytes memory extraData_);
    function gameIndex() external view returns (uint256);
    function gameType() external view returns (GameType gameType_);
    function getChallengerDuration(uint256 asOfTimestamp) external view returns (Duration duration_);
    function initialize() external payable;
    function isViableSignature(bytes32 childSignature) external view returns (bool isViableSignature_);
    function l1Head() external pure returns (Hash l1Head_);
    function l2SequenceNumber() external pure returns (uint256 l2SequenceNumber_);
    function minCreationTime() external view returns (Timestamp minCreationTime_);
    function opponentIndex() external view returns (uint64);
    function parentGame() external view returns (address parentGame_);
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
    ) external;
    function proveTrailFault(
        address payoutRecipient,
        uint64[2] memory co,
        uint256 proposedOutputFe,
        bytes memory blobCommitment,
        bytes memory kzgProof
    ) external;
    function proveValidity(address payoutRecipient, address l1HeadSource, uint64 childIndex, bytes memory encodedSeal)
    external;
    function provenAt(bytes32) external view returns (Timestamp);
    function prover(bytes32) external view returns (address);
    function pruneChildren(uint256 stepLimit) external returns (address);
    function resolve() external returns (GameStatus status_);
    function resolvedAt() external view returns (Timestamp);
    function rootClaim() external pure returns (Claim rootClaim_);
    function rootClaimByChainId(uint256) external pure returns (Claim rootClaim_);
    function signature() external view returns (bytes32 signature_);
    function status() external view returns (GameStatus);
    function validChildSignature() external view returns (bytes32);
    function verifyIntermediateOutput(
        uint64 outputNumber,
        uint256 outputFe,
        bytes memory blobCommitment,
        bytes memory kzgProof
    ) external returns (bool success);
    function version() external view returns (string memory);
    function wasRespectedGameTypeWhenCreated() external view returns (bool);
}
