// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title IZKVerifier
/// @notice Generic interface for ZK proof verification.
interface IZKVerifier is ISemver {
    /// @notice Returns a unique identifier for the verifier type, used to distinguish
    ///         between different ZK proof systems (e.g. SP1, Risc0, etc).
    function verifierType() external pure returns (string memory);

    /// @notice Verifies a ZK proof against public inputs. Reverts if invalid.
    ///
    /// @param _programId The program identifier (absolute prestate).
    /// @param _publicValues The ABI-encoded public values for verification.
    /// @param _proof The proof bytes.
    function verify(bytes32 _programId, bytes calldata _publicValues, bytes calldata _proof) external view;
}
