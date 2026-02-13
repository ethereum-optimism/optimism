// SPDX-License-Identifier: MIT
pragma solidity 0.8.24;

import { BondTransferFailed } from "src/dispute/lib/Errors.sol";

library KailuaLib {
    /// @notice Transfers ETH from the contract's balance to the recipient
    function pay(uint256 amount, address recipient) internal {
        (bool success,) = recipient.call{value: amount}(hex"");
        if (!success) revert BondTransferFailed();
    }

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
