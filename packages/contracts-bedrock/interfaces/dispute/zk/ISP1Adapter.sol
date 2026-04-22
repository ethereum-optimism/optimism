// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";
import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";

/// @title ISP1Adapter
/// @notice Interface for the SP1Adapter contract.
interface ISP1Adapter is IZKVerifier {
    /// @notice Thrown when the provided verifier address has no code.
    error SP1Adapter_InvalidVerifier();

    /// @notice Thrown when the provided proof system identifier is empty.
    error SP1Adapter_InvalidProofSystem();

    /// @notice Returns the address of the underlying SP1 verifier.
    function sp1Verifier() external view returns (ISP1Verifier sp1Verifier_);

    /// @notice Returns the proof system identifier supplied at construction time.
    function proofSystem() external view returns (string memory);

    /// @notice Constructor.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    /// @param _proofSystem The proof system identifier (e.g. "PLONK", "GROTH16").
    function __constructor__(ISP1Verifier _sp1Verifier, string memory _proofSystem) external;
}
