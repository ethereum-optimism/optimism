// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ISP1Verifier
/// @notice Interface for Succinct's SP1 verifier.
///         Derived from https://github.com/succinctlabs/sp1-contracts/tree/v6.0.0.
interface ISP1Verifier {
    /// @notice Verifies a proof with given public values and vkey.
    ///
    /// @param programVKey The verification key for the RISC-V program.
    /// @param publicValues The public values encoded as bytes.
    /// @param proofBytes The proof of the program execution the SP1 zkVM encoded as bytes.
    function verifyProof(
        bytes32 programVKey,
        bytes calldata publicValues,
        bytes calldata proofBytes
    ) external view;

    /// @notice The version string of the SP1 verifier (e.g. "v6.0.0").
    function VERSION() external pure returns (string memory);
}
