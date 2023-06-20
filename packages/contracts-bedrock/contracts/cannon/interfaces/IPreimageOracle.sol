// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { PreimageKey, PreimageOffset, PreimagePart, PreimageLength } from "../lib/CannonTypes.sol";

/// @title IPreimageOracle
/// @notice Interface for a preimage oracle.
interface IPreimageOracle {
    /// @notice Reads a preimage from the oracle.
    /// @param key The key of the preimage to read.
    /// @param offset The offset of the preimage to read.
    /// @return dat The preimage data.
    /// @return datLen The length of the preimage data.
    function readPreimage(PreimageKey key, PreimageOffset offset)
        external
        view
        returns (PreimagePart dat, PreimageLength datLen);

    /// @notice Computes and returns the key for a pre-image.
    /// @param preimage The pre-image.
    /// @return key The pre-image key.
    function computePreimageKey(bytes calldata preimage) external pure returns (PreimageKey key);

    /// @notice Prepares a preimage to be read by keccak256 key, starting at
    ///         the given offset and up to 32 bytes (clipped at preimage length, if out of data).
    /// @param partOffset The offset of the preimage to read.
    /// @param preimage The preimage data.
    function loadKeccak256PreimagePart(PreimageOffset partOffset, bytes calldata preimage) external;
}
