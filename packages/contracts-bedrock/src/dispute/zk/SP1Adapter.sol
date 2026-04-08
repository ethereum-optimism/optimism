// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { ISP1Verifier } from "interfaces/vendor/ISP1Verifier.sol";
import { IZKVerifier } from "interfaces/dispute/zk/IZKVerifier.sol";

/// @title SP1Adapter
/// @notice Adapter that wraps an SP1 verifier behind the IZKVerifier interface.
///         Deployed as a singleton (not proxied), following the MIPS.sol pattern.
contract SP1Adapter is IZKVerifier {
    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice Address of the actual SP1 verifier.
    ISP1Verifier public immutable sp1Verifier;

    /// @notice Constructs the SP1Adapter.
    ///
    /// @param _sp1Verifier The SP1 verifier contract.
    constructor(ISP1Verifier _sp1Verifier) {
        sp1Verifier = _sp1Verifier;
    }

    /// @notice Returns a verifier type identifier combining "SP1-" with the
    ///         verifier's version string.
    function verifierType() external view returns (string memory) {
        return string(abi.encodePacked("SP1-", sp1Verifier.VERSION()));
    }

    /// @notice Verifies an SP1 proof. Reverts if invalid.
    ///
    /// @param _programId The program identifier (absolute prestate).
    /// @param _publicValues The ABI-encoded public values for verification.
    /// @param _proof The proof bytes.
    function verify(bytes32 _programId, bytes calldata _publicValues, bytes calldata _proof) external view {
        sp1Verifier.verifyProof(_programId, _publicValues, _proof);
    }
}
