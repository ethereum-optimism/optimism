// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IRiscZeroVerifier } from "interfaces/vendor/IRiscZeroVerifier.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title RiscZeroAdapter
/// @notice Adapter that wraps a RISC Zero Groth16 verifier behind the IZKVerifier interface.
///         Unlike SP1 (where the verifier interface maps almost 1:1), RISC Zero uses a different
///         verification interface that requires hashing public values into a journal digest and
///         reordering arguments.
///         Deployed as a singleton (not proxied), following the MIPS.sol pattern.
contract RiscZeroAdapter is IZKVerifier {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Address of the actual RISC Zero verifier.
    IRiscZeroVerifier internal immutable RISC_ZERO_VERIFIER; // nosemgrep: sol-safety-no-immutable-variables

    /// @notice Constructs the RiscZeroAdapter.
    ///
    /// @param _riscZeroVerifier The RISC Zero verifier contract.
    constructor(IRiscZeroVerifier _riscZeroVerifier) {
        RISC_ZERO_VERIFIER = _riscZeroVerifier;
    }

    /// @notice Returns the address of the underlying RISC Zero verifier.
    function riscZeroVerifier() external view returns (IRiscZeroVerifier riscZeroVerifier_) {
        riscZeroVerifier_ = RISC_ZERO_VERIFIER;
    }

    /// @notice Returns a verifier type identifier combining "RISC_ZERO-" with the
    ///         verifier's version string.
    function verifierType() external view returns (string memory) {
        return string(abi.encodePacked("RISC_ZERO-", RISC_ZERO_VERIFIER.VERSION()));
    }

    /// @notice Verifies a ZK proof by translating arguments for the RISC Zero verifier.
    /// @dev RISC Zero expects the SHA-256 digest of the public values (journal) rather than
    ///      the raw values. The off-chain RISC Zero program must produce a journal whose
    ///      SHA-256 hash matches the ABI encoding of the public values constructed by
    ///      ZKDisputeGame.prove().
    ///
    /// @param _programId The program identifier (maps to RISC Zero's imageId).
    /// @param _publicValues The ABI-encoded public values, hashed to produce journalDigest.
    /// @param _proof The Groth16 proof bytes (maps to RISC Zero's seal).
    function verify(bytes32 _programId, bytes calldata _publicValues, bytes calldata _proof) external view {
        bytes32 journalDigest = sha256(_publicValues);
        RISC_ZERO_VERIFIER.verify(_proof, _programId, journalDigest);
    }
}
