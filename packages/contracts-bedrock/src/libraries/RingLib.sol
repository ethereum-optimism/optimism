// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

uint32 constant RING_BITS = 16;
uint32 constant RING_SIZE = 65536;

/// @title RingLib
/// @notice Library for managing a ring buffer
/// @author philogy <https://github.com/philogy>
library RingLib {
    /// @notice Set a value in the ring buffer at a specific index
    /// @param _ring The ring buffer to modify
    /// @param _index The index at which to set the value
    /// @param _hash The value to set in the ring buffer at the specified index
    function set(bytes32[RING_SIZE] storage _ring, uint64 _index, bytes32 _hash) internal {
        assembly {
            let relIndex := and(_index, sub(shl(RING_BITS, 1), 1))
            sstore(add(_ring.slot, relIndex), _hash)
        }
    }

    /// @notice Retrieve a value from the ring buffer at a specific index
    /// @param _ring The ring buffer to retrieve from
    /// @param _index The index at which to retrieve the value
    /// @return hash_ The value retrieved from the ring buffer at the specified index
    function get(bytes32[RING_SIZE] storage _ring, uint64 _index) internal view returns (bytes32 hash_) {
        assembly {
            hash_ := sload(add(_ring.slot, and(_index, sub(shl(RING_BITS, 1), 1))))
        }
    }
}
