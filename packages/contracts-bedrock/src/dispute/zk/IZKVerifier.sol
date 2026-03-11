// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IZKVerifier
/// @notice Generic interface for ZK proof verification.
interface IZKVerifier {
    /// @notice Verifies a ZK proof against public inputs. Reverts if invalid.
    ///
    /// @param _publicInput The ABI-encoded public inputs for verification.
    /// @param _proof The proof bytes.
    function verify(bytes calldata _publicInput, bytes calldata _proof) external view;
}
