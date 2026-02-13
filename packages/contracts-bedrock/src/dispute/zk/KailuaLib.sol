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

import "../lib/Errors.sol";
import "@openzeppelin/contracts/utils/Timers.sol";

// 0xd36871fd
/// @notice Thrown when a blacklisted address attempts to interact with the game.
error Blacklisted(address source, address expected);

// 0x9d3e7d24
/// @notice Thrown when a child from an unknown source appends itself to a tournament
error UnknownGame();

// 0x8b1dfa22
/// @notice Thrown when eliminating an already removed child
error AlreadyEliminated();

// 0x2c06a364
/// @notice Thrown when a proof is submitted for an already proven game
error AlreadyProven();

// 0xa506d334
/// @notice Thrown when a resolution is attempted for an unproven claim
error NotProven();

// 0x5e22e582
/// @notice Thrown when resolving a faulty proposal
error ProvenFaulty();

// 0xf2a87d5e
/// @notice Thrown when pruning is attempted with no children
error NotProposed();

// 0x7412124e
/// @notice Thrown when proving is attempted with two agreeing outputs
error NoConflict();

// 0x9276ab5a
/// @notice Thrown when proposing before the minimum creation time
error ProposalGapRemaining(uint256 currentTime, uint256 minCreationTime);

// 0x1434391f
/// @notice Thrown when a blob hash is missing
error BlobHashMissing(uint256 index, uint256 count);

// 0x19e3a1dc
/// @notice Occurs when the duplication counter is wrong
error InvalidDuplicationCounter();

// 0xeaa0996e
/// @notice Occurs when the anchored game block number is different
/// @param anchored The L2 block number of the anchored game
/// @param initialized This game's l2 block number
error BlockNumberMismatch(uint256 anchored, uint256 initialized);

// 0x627fad6e
/// @notice Occurs when a proposer attempts to extend the chain before the vanguard
/// @param parentGame The address of the parent proposal being extended
error VanguardError(address parentGame);

// 0x428e0b92
/// @notice Thrown when a non-factory owner calls an owner-only function.
error NotFactoryOwner();

/// @notice Error for when a deposit or withdrawal is to a bad target.
error BadTarget();

library KailuaPayLib {
    /// @notice Transfers ETH from the contract's balance to the recipient
    function pay(uint256 amount, address recipient) internal {
        (bool success,) = recipient.call{value: amount}(hex"");
        if (!success) revert BondTransferFailed();
    }
}

library KailuaKZGLib {
    /// @notice The KZG commitment version
    bytes32 internal constant KZG_COMMITMENT_VERSION =
        bytes32(0x0100000000000000000000000000000000000000000000000000000000000000);

    /// @notice The modular exponentiation precompile
    address internal constant MOD_EXP = address(0x05);

    /// @notice The point evaluation precompile
    address internal constant KZG = address(0x0a);

    /// @notice The expected result from the point evaluation precompile
    bytes32 internal constant KZG_RESULT = keccak256(abi.encodePacked(FIELD_ELEMENTS_PER_BLOB, BLS_MODULUS));

    /// @notice Scalar field modulus of BLS12-381
    uint256 internal constant BLS_MODULUS =
        52435875175126190479447740508185965837690552500527637822603658699938581184513;

    /// @notice The base root of unity for indexing blob field elements
    uint256 internal constant ROOT_OF_UNITY =
        39033254847818212395286706435128746857159659164139250548781411570340225835782;

    /// @notice The po2 for the number of field elements in a single blob
    uint256 internal constant FIELD_ELEMENTS_PER_BLOB_PO2 = 12;

    /// @notice The number of field elements in a single blob
    uint256 internal constant FIELD_ELEMENTS_PER_BLOB = uint64(1 << FIELD_ELEMENTS_PER_BLOB_PO2);

    /// @notice The index of the blob containing the FE at the provided offset
    function blobIndex(uint256 outputOffset) internal pure returns (uint256 index) {
        index = outputOffset / FIELD_ELEMENTS_PER_BLOB;
    }

    /// @notice The index of the FE at the provided offset in the blob that contains it
    function fieldElementIndex(uint256 outputOffset) internal pure returns (uint32 position) {
        position = uint32(outputOffset % FIELD_ELEMENTS_PER_BLOB);
    }

    /// @notice The versioned KZG hash of the provided blob commitment
    function versionedKZGHash(bytes calldata blobCommitment) internal pure returns (bytes32 hash) {
        require(blobCommitment.length == 48);
        hash = ((sha256(blobCommitment) << 8) >> 8) | KZG_COMMITMENT_VERSION;
    }

    /// @notice The mapped FE corresponding to the input hash
    function hashToFe(bytes32 hash) internal pure returns (uint256 fe) {
        fe = uint256(hash) % BLS_MODULUS;
    }

    /// @notice Returns true iff the proof shows that the FE is part of the blob at the provided position
    function verifyKZGBlobProof(
        bytes32 versionedBlobHash,
        uint32 index,
        uint256 value,
        bytes calldata blobCommitment,
        bytes calldata proof
    ) internal returns (bool success) {
        uint256 rootOfUnity = modExp(reverseBits(index));
        // Byte range	Name	        Description
        // [0:32]	    versioned_hash	Reference to a blob in the execution layer.
        // [32:64]	    x	            x-coordinate at which the blob is being evaluated.
        // [64:96]	    y	            y-coordinate at which the blob is being evaluated.
        // [96:144]	    commitment	    Commitment to the blob being evaluated.
        // [144:192]	proof	        Proof associated with the commitment.
        bytes memory kzgCallData = abi.encodePacked(versionedBlobHash, rootOfUnity, value, blobCommitment, proof);
        // The precompile will reject non-canonical field elements (i.e. value must be less than BLS_MODULUS).
        (bool _success, bytes memory kzgResult) = KZG.call(kzgCallData);
        // Validate the precompile response
        require(keccak256(kzgResult) == KZG_RESULT);
        // Return the result
        success = _success;
    }

    /// @notice Calls the modular exponentiation precompile with a fixed base and modulus
    function modExp(uint256 exponent) internal returns (uint256 result) {
        bytes memory modExpData =
            abi.encodePacked(uint256(32), uint256(32), uint256(32), ROOT_OF_UNITY, exponent, BLS_MODULUS);
        (bool success, bytes memory mexpResult) = MOD_EXP.call(modExpData);
        require(success);
        result = uint256(bytes32(mexpResult));
    }

    /// @notice Reverses the bits of the input index
    function reverseBits(uint32 index) internal pure returns (uint256 result) {
        for (uint256 i = 0; i < FIELD_ELEMENTS_PER_BLOB_PO2; i++) {
            result <<= 1;
            result |= ((1 << i) & index) >> i;
        }
    }
}
