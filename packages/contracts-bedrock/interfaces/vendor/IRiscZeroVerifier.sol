// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IRiscZeroVerifier
/// @notice Interface for the RISC Zero Groth16 verifier.
///         Derived from https://github.com/risc0/risc0-ethereum.
interface IRiscZeroVerifier {
    /// @notice Verifies a RISC Zero proof.
    ///
    /// @param seal The Groth16 proof bytes (seal).
    /// @param imageId The identifier of the zkVM guest program.
    /// @param journalDigest The SHA-256 digest of the journal (public outputs).
    function verify(bytes calldata seal, bytes32 imageId, bytes32 journalDigest) external view;

    /// @notice The version string of the RISC Zero verifier.
    function VERSION() external pure returns (string memory);
}
