// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import "src/cannon/libraries/CannonErrors.sol";

library MIPSMemory {
    /// @notice Reads a 32-bit value from memory.
    /// @param _memRoot The current memory root
    /// @param _addr The address to read from.
    /// @param _proofOffset The offset of the memory proof in calldata.
    /// @return out_ The hashed MIPS state.
    function readMem(bytes32 _memRoot, uint32 _addr, uint256 _proofOffset) internal pure returns (uint32 out_) {
        bool valid;
        (out_, valid) = readMemUnchecked(_memRoot, _addr, _proofOffset);
        if (!valid) {
            revert InvalidMemoryProof();
        }
    }

    /// @notice Reads a 32-bit value from memory.
    /// @param _memRoot The current memory root
    /// @param _addr The address to read from.
    /// @param _proofOffset The offset of the memory proof in calldata.
    /// @return out_ The hashed MIPS state.
    ///         valid_ Whether the proof is valid.
    function readMemUnchecked(
        bytes32 _memRoot,
        uint32 _addr,
        uint256 _proofOffset
    )
        internal
        pure
        returns (uint32 out_, bool valid_)
    {
        unchecked {
            validateMemoryProofAvailability(_proofOffset);
            assembly {
                // Validate the address alignement.
                if and(_addr, 3) { revert(0, 0) }

                // Load the leaf value.
                let leaf := calldataload(_proofOffset)
                _proofOffset := add(_proofOffset, 32)

                // Convenience function to hash two nodes together in scratch space.
                function hashPair(a, b) -> h {
                    mstore(0, a)
                    mstore(32, b)
                    h := keccak256(0, 64)
                }

                // Start with the leaf node.
                // Work back up by combining with siblings, to reconstruct the root.
                let path := shr(5, _addr)
                let node := leaf
                for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
                    let sibling := calldataload(_proofOffset)
                    _proofOffset := add(_proofOffset, 32)
                    switch and(shr(i, path), 1)
                    case 0 { node := hashPair(node, sibling) }
                    case 1 { node := hashPair(sibling, node) }
                }

                // Verify the root matches.
                valid_ := eq(node, _memRoot)
                if valid_ {
                    // Bits to shift = (32 - 4 - (addr % 32)) * 8
                    let shamt := shl(3, sub(sub(32, 4), and(_addr, 31)))
                    out_ := and(shr(shamt, leaf), 0xFFffFFff)
                }
            }
        }
    }

    /// @notice Writes a 32-bit value to memory.
    ///         This function first overwrites the part of the leaf.
    ///         Then it recomputes the memory merkle root.
    /// @param _addr The address to write to.
    /// @param _proofOffset The offset of the memory proof in calldata.
    /// @param _val The value to write.
    /// @return newMemRoot_ The new memory root after modification
    function writeMem(uint32 _addr, uint256 _proofOffset, uint32 _val) internal pure returns (bytes32 newMemRoot_) {
        unchecked {
            validateMemoryProofAvailability(_proofOffset);
            assembly {
                // Validate the address alignement.
                if and(_addr, 3) { revert(0, 0) }

                // Load the leaf value.
                let leaf := calldataload(_proofOffset)
                let shamt := shl(3, sub(sub(32, 4), and(_addr, 31)))

                // Mask out 4 bytes, and OR in the value
                leaf := or(and(leaf, not(shl(shamt, 0xFFffFFff))), shl(shamt, _val))
                _proofOffset := add(_proofOffset, 32)

                // Convenience function to hash two nodes together in scratch space.
                function hashPair(a, b) -> h {
                    mstore(0, a)
                    mstore(32, b)
                    h := keccak256(0, 64)
                }

                // Start with the leaf node.
                // Work back up by combining with siblings, to reconstruct the root.
                let path := shr(5, _addr)
                let node := leaf
                for { let i := 0 } lt(i, 27) { i := add(i, 1) } {
                    let sibling := calldataload(_proofOffset)
                    _proofOffset := add(_proofOffset, 32)
                    switch and(shr(i, path), 1)
                    case 0 { node := hashPair(node, sibling) }
                    case 1 { node := hashPair(sibling, node) }
                }

                newMemRoot_ := node
            }
            return newMemRoot_;
        }
    }

    /// @notice Verifies a memory proof.
    /// @param _memRoot The expected memory root
    /// @param _addr The _addr proven.
    /// @param _proofOffset The offset of the memory proof in calldata.
    /// @return valid_ True iff it is a valid proof.
    function isValidProof(bytes32 _memRoot, uint32 _addr, uint256 _proofOffset) internal pure returns (bool valid_) {
        (, valid_) = readMemUnchecked(_memRoot, _addr, _proofOffset);
    }

    /// @notice Computes the offset of a memory proof in the calldata.
    /// @param _proofDataOffset The offset of the set of all memory proof data within calldata (proof.offset)
    ///     Equal to the offset of the first memory proof (at _proofIndex 0).
    /// @param _proofIndex The index of the proof in the calldata.
    /// @return offset_ The offset of the memory proof at the given _proofIndex in the calldata.
    function memoryProofOffset(uint256 _proofDataOffset, uint8 _proofIndex) internal pure returns (uint256 offset_) {
        unchecked {
            // A proof of 32 bit memory, with 32-byte leaf values, is (32-5)=27 bytes32 entries.
            // And the leaf value itself needs to be encoded as well: (27 + 1) = 28 bytes32 entries.
            offset_ = _proofDataOffset + (uint256(_proofIndex) * (28 * 32));
            return offset_;
        }
    }

    /// @notice Validates that enough calldata is available to hold a full memory proof at the given offset
    /// @param _proofStartOffset The index of the first byte of the target memory proof in calldata
    function validateMemoryProofAvailability(uint256 _proofStartOffset) internal pure {
        unchecked {
            uint256 s = 0;
            assembly {
                s := calldatasize()
            }
            // A memory proof consists of 28 bytes32 values - verify we have enough calldata
            require(s >= (_proofStartOffset + 28 * 32), "check that there is enough calldata");
        }
    }
}
