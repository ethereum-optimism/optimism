// SPDX-License-Identifier: MIT
pragma solidity 0.7.6;

/// @title IPreimageOracle
/// @notice Interface for a preimage oracle.
interface IPreimageOracle {
    /// @notice Reads a preimage from the oracle.
    /// @param _key The key of the preimage to read.
    /// @param _offset The offset of the preimage to read.
    /// @return dat_ The preimage data.
    /// @return datLen_ The length of the preimage data.
    function readPreimage(bytes32 _key, uint256 _offset)
        external
        view
        returns (bytes32 dat_, uint256 datLen_);

    /// @notice Computes and returns the key for a pre-image.
    /// @param _preimage The pre-image.
    /// @return key_ The pre-image key.
    function computePreimageKey(bytes calldata _preimage) external pure returns (bytes32 key_);

    /// @notice Prepares a preimage to be read by keccak256 key, starting at
    ///         the given offset and up to 32 bytes (clipped at preimage length, if out of data).
    /// @param _partOffset The offset of the preimage to read.
    /// @param _preimage The preimage data.
    function loadKeccak256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external;
}
