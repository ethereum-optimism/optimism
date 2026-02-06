// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title SP1 Verifier Interface
/// @author Succinct Labs
/// @notice This contract is the interface for the SP1 Verifier.
interface ISP1Verifier {
    /// @notice Verifies a proof with given public values and vkey.
    /// @dev It is expected that the first 4 bytes of proofBytes must match the first 4 bytes of
    /// target verifier's VERIFIER_HASH.
    /// @param _programVKey The verification key for the RISC-V program.
    /// @param _publicValues The public values encoded as bytes.
    /// @param _proofBytes The proof of the program execution the SP1 zkVM encoded as bytes.
    function verifyProof(bytes32 _programVKey, bytes calldata _publicValues, bytes calldata _proofBytes) external view;
}
