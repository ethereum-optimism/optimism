// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title PreimageKeyLib
/// @notice Shared utilities for localizing local keys in the preimage oracle.
library PreimageKeyLib {
    /// @notice Generates a context-specific local key for the given local data identifier.
    /// @dev See `localize` for a description of the localization operation.
    /// @param _ident The identifier of the local data. [0, 32) bytes in size.
    /// @return key_ The context-specific local key.
    function localizeIdent(uint256 _ident) internal view returns (bytes32 key_) {
        assembly {
            // Set the type byte in the given identifier to `1` (Local). We only care about
            // the [1, 32) bytes in this value.
            key_ := or(shl(248, 1), and(_ident, not(shl(248, 0xFF))))
        }
        key_ = localize(key_);
    }

    /// @notice Localizes a given local data key for the caller's context.
    /// @dev The localization operation is defined as:
    ///      localize(k) = H(k .. sender) & ~(0xFF << 248) | (0x01 << 248)
    ///      where H is the Keccak-256 hash function.
    /// @param _key The local data key to localize.
    /// @return localizedKey_ The localized local data key.
    function localize(bytes32 _key) internal view returns (bytes32 localizedKey_) {
        assembly {
            // Store the local data key and caller next to each other in memory for hashing.
            mstore(0, _key)
            mstore(0x20, caller())
            // Localize the key with the above `localize` operation.
            localizedKey_ := or(and(keccak256(0, 0x40), not(shl(248, 0xFF))), shl(248, 1))
        }
    }

    /// @notice Computes and returns the key for a global keccak pre-image.
    /// @param _preimage The pre-image.
    /// @return key_ The pre-image key.
    function keccak256PreimageKey(bytes memory _preimage) internal pure returns (bytes32 key_) {
        assembly {
            // Grab the size of the `_preimage`
            let size := mload(_preimage)

            // Compute the pre-image keccak256 hash (aka the pre-image key)
            let h := keccak256(add(_preimage, 0x20), size)

            // Mask out prefix byte, replace with type 2 byte
            key_ := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }
    }
}
