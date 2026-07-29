// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title SP1PlonkAdapter
/// @notice Adapter that wraps an SP1 PLONK verifier behind the IZKVerifier interface.
///         Deployed as a singleton (not proxied), following the MIPS.sol pattern.
contract SP1PlonkAdapter is IZKVerifier {
    /// @notice Thrown when the provided verifier address has no code.
    error SP1PlonkAdapter_InvalidVerifier();

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Address of the actual SP1 verifier.
    ISP1Verifier internal immutable SP1_VERIFIER; // nosemgrep: sol-safety-no-immutable-variables

    /// @notice Constructs the SP1PlonkAdapter.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    constructor(ISP1Verifier _sp1Verifier) {
        if (address(_sp1Verifier).code.length == 0) revert SP1PlonkAdapter_InvalidVerifier();
        SP1_VERIFIER = _sp1Verifier;
    }

    /// @notice Returns the address of the underlying SP1 verifier.
    function sp1Verifier() external view returns (ISP1Verifier sp1Verifier_) {
        sp1Verifier_ = SP1_VERIFIER;
    }

    /// @notice Returns a verifier type identifier combining "SP1-PLONK-" with the
    ///         verifier's version string.
    function verifierType() external view returns (string memory) {
        return string(abi.encodePacked("SP1-PLONK-", SP1_VERIFIER.VERSION()));
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
