// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IZKVerifier
/// @notice Generic interface for ZK proof verifiers. Implementors may wrap any ZK proving system
///         (e.g. SP1, RISC Zero, Groth16) behind this interface so that ZKDisputeGame remains
///         verifier-agnostic.
interface IZKVerifier {
    /// @notice Verifies a ZK proof against a program identifier and its public outputs.
    ///
    /// @param _absolutePrestate A 32-byte identifier for the ZK program whose execution is being
    ///                          verified. For SP1 this is the aggregation program verification key.
    ///                          For other schemes it is their equivalent program identifier.
    ///
    /// @param _publicValues      ABI-encoded public outputs committed to by the proof. For
    ///                          ZKDisputeGame these are the ABI-encoded ZKPublicValues struct.
    ///
    /// @param _proofBytes        The raw proof bytes produced by the prover.
    ///
    /// @dev Implementations MUST revert if the proof is invalid.
    function verify(bytes32 _absolutePrestate, bytes calldata _publicValues, bytes calldata _proofBytes) external view;
}
