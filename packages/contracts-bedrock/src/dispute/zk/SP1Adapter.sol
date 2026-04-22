// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title SP1Adapter
/// @notice Adapter that wraps an SP1 verifier behind the IZKVerifier interface.
///         The proof system (e.g. "PLONK", "GROTH16") is supplied at construction
///         time so the same adapter can be reused across SP1 verifier variants.
///         Deployed as a singleton (not proxied), following the MIPS.sol pattern.
contract SP1Adapter is IZKVerifier {
    /// @notice Thrown when the provided verifier address has no code.
    error SP1Adapter_InvalidVerifier();

    /// @notice Thrown when the provided proof system identifier is empty.
    error SP1Adapter_InvalidProofSystem();

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Address of the actual SP1 verifier.
    ISP1Verifier internal immutable SP1_VERIFIER; // nosemgrep: sol-safety-no-immutable-variables

    /// @notice Identifier of the proof system implemented by the underlying verifier
    ///         (e.g. "PLONK", "GROTH16"). Used to build `verifierType`. Set once in
    ///         the constructor and never modified afterwards; stored in storage
    ///         because Solidity 0.8.15 does not support immutable strings.
    string internal proofSystem_;

    /// @notice Constructs the SP1Adapter.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    /// @param _proofSystem The proof system identifier (e.g. "PLONK", "GROTH16").
    constructor(ISP1Verifier _sp1Verifier, string memory _proofSystem) {
        if (address(_sp1Verifier).code.length == 0) revert SP1Adapter_InvalidVerifier();
        if (bytes(_proofSystem).length == 0) revert SP1Adapter_InvalidProofSystem();
        SP1_VERIFIER = _sp1Verifier;
        proofSystem_ = _proofSystem;
    }

    /// @notice Returns the address of the underlying SP1 verifier.
    function sp1Verifier() external view returns (ISP1Verifier sp1Verifier_) {
        sp1Verifier_ = SP1_VERIFIER;
    }

    /// @notice Returns the proof system identifier supplied at construction time.
    function proofSystem() external view returns (string memory) {
        return proofSystem_;
    }

    /// @notice Returns a verifier type identifier combining "SP1-", the proof system
    ///         identifier, and the underlying verifier's version string
    ///         (e.g. "SP1-PLONK-v6.0.0").
    function verifierType() external view returns (string memory) {
        return string(abi.encodePacked("SP1-", proofSystem_, "-", SP1_VERIFIER.VERSION()));
    }

    /// @notice Verifies an SP1 proof. Reverts if invalid.
    ///
    /// @param _programId The program identifier (absolute prestate).
    /// @param _publicValues The ABI-encoded public values for verification.
    /// @param _proof The proof bytes.
    function verify(bytes32 _programId, bytes calldata _publicValues, bytes calldata _proof) external view {
        SP1_VERIFIER.verifyProof(_programId, _publicValues, _proof);
    }
}
