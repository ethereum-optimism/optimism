// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/// @title PreimageOracle
/// @notice abc123
contract PreimageOracle {
    /// @notice A `PreimageKeyKind` is the type of preimage key that is being queried.
    /// @dev https://github.com/ethereum-optimism/optimism/blob/develop/specs/fault-proof.md#pre-image-key-types
    enum PreimageKeyKind {
        /// @notice The zero prefix is illegal. This ensures all pre-image keys are non-zero, enabling
        ///         storage lookup optimizations and avoiding easy mistakes with invalid zeroed keys
        ///         in the EVM.
        ZERO,
        /// @notice The local key is information specific to a given dispute. Keys with this type
        ///         map to data that is only valid within the context of a given dispute game proxy,
        ///         embedded in bytes `[1, 21)` of the key.
        LOCAL_DATA,
        /// @notice Global keccak256 preimage data. Keys with this type map to data that exists
        ///         independent of context.
        PREIMAGE,
        /// @notice Global generic key. This is reserved to allow for up to 0xFF preimage types
        ///         without fault proof VM redeployments.
        GLOBAL
    }

    function readPreimage(
        bytes32 key,
        uint256 offset
    ) external view returns (bytes32 dat, uint256 datLen) {
        (bytes32 preimagePartOkSlot, bytes32 preimageLengthsSlot, bytes32 preimagePartsSlot) =
            _getKeyKindMappingSlots(key);
        assembly {
            // Loads a value from a mapping. Only works for mappings with a fixed-size
            // key <= 32 bytes in size and a value that is fixed-size and <= 32 bytes
            // in size.
            function loadFromMapping(k, mappingSlot) -> val {
                // Compute the slot of the value & load it
                mstore(0x00, k)
                mstore(0x20, mappingSlot)
                val := sload(keccak256(0x00, 0x40))
            }

            // Loads a value from a nested mapping. Only works for nested mappings
            // with two fixed-size keys <= 32 bytes in size and a final value that
            // is fixed-size and <= 32 bytes in size.
            function loadFromNestedMapping(keyA, keyB, mappingSlot) -> val {
                // Compute the slot of the nested mapping
                mstore(0x00, keyA)
                mstore(0x20, mappingSlot)
                let nestedSlot := keccak256(0x00, 0x40)
                // Compute the slot of the value & load it
                mstore(0x00, keyB)
                mstore(0x20, nestedSlot)
                val := sload(keccak256(0x00, 0x40))
            }

            // The preimage part must exist to be read.
            if iszero(loadFromNestedMapping(key, offset, preimagePartOkSlot)) {
                revert(0, 0)
            }

            datLen := 32
            let length := loadFromMapping(key, preimageLengthsSlot)
            if iszero(gt(add(length, 8), add(offset, 32))) {
                datLen := sub(add(length, 8), offset)
            }
            dat := loadFromNestedMapping(key, offset, preimagePartsSlot)
        }
    }

    // TODO(CLI-4104):
    // we need to mix-in the ID of the dispute for local-type keys to avoid collisions,
    // and restrict local pre-image insertion to the dispute-managing contract.
    // For now we permit anyone to write any pre-image unchecked, to make testing easy.
    // This method is DANGEROUS. And NOT FOR PRODUCTION.
    function cheat(
        uint256 partOffset,
        bytes32 key,
        bytes32 part,
        uint256 size
    ) external {
        (bytes32 preimagePartOkSlot, bytes32 preimageLengthsSlot, bytes32 preimagePartsSlot) =
            _getKeyKindMappingSlots(key);
        assembly {
            // Stores a value in a mapping. Only works for mappings with a fixed-size
            // key <= 32 bytes in size and a value that is fixed-size and <= 32 bytes
            // in size.
            function storeInMapping(k, v, mappingSlot) {
                // Compute the slot of the value & load it
                mstore(0x00, k)
                mstore(0x20, mappingSlot)
                sstore(keccak256(0x00, 0x40), v)
            }

            // Stores a value in a nested mapping. Only works for nested mappings
            // with two fixed-size keys <= 32 bytes in size and a final value that
            // is fixed-size and <= 32 bytes in size.
            function storeInNestedMapping(keyA, keyB, val, mappingSlot) {
                // Compute the slot of the nested mapping
                mstore(0x00, keyA)
                mstore(0x20, mappingSlot)
                let nestedSlot := keccak256(0x00, 0x40)
                // Compute the slot of the value & load it
                mstore(0x00, keyB)
                mstore(0x20, nestedSlot)
                sstore(keccak256(0x00, 0x40), val)
            }

            storeInNestedMapping(key, partOffset, true, preimagePartOkSlot)
            storeInNestedMapping(key, partOffset, part, preimagePartsSlot)
            storeInMapping(key, size, preimageLengthsSlot)
        }
    }

    // loadKeccak256PreimagePart prepares the pre-image to be read by keccak256 key,
    // starting at the given offset, up to 32 bytes (clipped at preimage length, if out of data).
    function loadKeccak256PreimagePart(uint256 partOffset, bytes calldata preimage) external {
        uint256 size;
        bytes32 key;
        bytes32 part;
        assembly {
            // len(sig) + len(partOffset) + len(preimage offset) = 4 + 32 + 32 = 0x44
            size := calldataload(0x44)
            // revert if part offset >= size+8 (i.e. parts must be within bounds)
            if iszero(lt(partOffset, add(size, 8))) {
                revert(0, 0)
            }
            // we leave solidity slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80
            // put size as big-endian uint64 at start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)
            // copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, preimage.offset, size)
            // Note that it includes the 8-byte big-endian uint64 length prefix.
            // this will be zero-padded at the end, since memory at end is clean.
            part := mload(add(sub(ptr, 8), partOffset))
            let h := keccak256(ptr, size) // compute preimage keccak256 hash
            // mask out prefix byte, replace with type 2 byte
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }

        (bytes32 preimagePartOkSlot, bytes32 preimageLengthsSlot, bytes32 preimagePartsSlot) =
            _getKeyKindMappingSlots(key);

        assembly {
            // Stores a value in a mapping. Only works for mappings with a fixed-size
            // key <= 32 bytes in size and a value that is fixed-size and <= 32 bytes
            // in size.
            function storeInMapping(k, v, mappingSlot) {
                // Compute the slot of the value & load it
                mstore(0x00, k)
                mstore(0x20, mappingSlot)
                sstore(keccak256(0x00, 0x40), v)
            }

            // Stores a value in a nested mapping. Only works for nested mappings
            // with two fixed-size keys <= 32 bytes in size and a final value that
            // is fixed-size and <= 32 bytes in size.
            function storeInNestedMapping(keyA, keyB, val, mappingSlot) {
                // Compute the slot of the nested mapping
                mstore(0x00, keyA)
                mstore(0x20, mappingSlot)
                let nestedSlot := keccak256(0x00, 0x40)
                // Compute the slot of the value & load it
                mstore(0x00, keyB)
                mstore(0x20, nestedSlot)
                sstore(keccak256(0x00, 0x40), val)
            }

            storeInNestedMapping(key, partOffset, true, preimagePartOkSlot)
            storeInNestedMapping(key, partOffset, part, preimagePartsSlot)
            storeInMapping(key, size, preimageLengthsSlot)
        }
    }

    // loadLocalBootData prepares the boot data for a game to be read by the local key,
    // starting at the given offset, up to 32 bytes (clipped at boot data length, if out of data).
    function loadLocalBootData(
        address _game,
        uint256 _partOffset,
        bytes calldata _bootData
    ) external {
        // Boot data is always a local key.
        (bytes32 preimagePartOkSlot, bytes32 preimageLengthsSlot, bytes32 preimagePartsSlot) =
            _getKeyKindMappingSlots(bytes32(uint256(1) << 248));
        assembly {
            // Stores a value in a mapping. Only works for mappings with a fixed-size
            // key <= 32 bytes in size and a value that is fixed-size and <= 32 bytes
            // in size.
            function storeInMapping(k, v, mappingSlot) {
                // Compute the slot of the value & load it
                mstore(0x00, k)
                mstore(0x20, mappingSlot)
                sstore(keccak256(0x00, 0x40), v)
            }

            // Stores a value in a nested mapping. Only works for nested mappings
            // with two fixed-size keys <= 32 bytes in size and a final value that
            // is fixed-size and <= 32 bytes in size.
            function storeInNestedMapping(keyA, keyB, val, mappingSlot) {
                // Compute the slot of the nested mapping
                mstore(0x00, keyA)
                mstore(0x20, mappingSlot)
                let nestedSlot := keccak256(0x00, 0x40)
                // Compute the slot of the value & load it
                mstore(0x00, keyB)
                mstore(0x20, nestedSlot)
                sstore(keccak256(0x00, 0x40), val)
            }

            // Load the length of the `_bootData` contents.
            // Stored at len(sig) + len(_game) + 0x0C + len(_partOffset) = 4 + 32 + 20 + 12 + 32 = 0x64
            // in calldata.
            let size := calldataload(0x64)

            // Revert if part offset >= size + 8 (i.e. parts must be within bounds)
            if iszero(lt(_partOffset, add(size, 8))) {
                revert(0, 0)
            }

            // We leave solidity slots 0x40 and 0x60 untouched, and everything after as scratch-memory.
            let ptr := 0x80

            // Store size as big-endian uint64 at start of pre-image
            mstore(ptr, shl(192, size))

            // Copy preimage payload into memory so we can hash and read it.
            calldatacopy(add(ptr, 8), _bootData.offset, size)
            // Note that the part includes the 8-byte big-endian uint64 length prefix.
            // This will be zero-padded at the end, since memory past it is clean.
            let part := mload(add(ptr, _partOffset))

            // The key of the boot data part is `1 << 248 | _game << 88 | localKey`
            let key := or(or(shl(248, 1), shl(88, _game)), 1)

            // Store a flag in the `localDataPartOk` mapping to indicate that the part exists.
            storeInNestedMapping(key, _partOffset, true, preimagePartOkSlot)
            // Store the part in the `localDataParts` mapping
            storeInNestedMapping(key, _partOffset, part, preimagePartsSlot)
            // Store the size in the `localDataLengths` mapping
            storeInMapping(key, size, preimageLengthsSlot)
        }
    }

    /// @notice Returns the mapping storage slots for the given key kind.
    /// @param _key The key to get the mapping slots for.
    /// @return partOk_ The slot of the mapping that stores whether a preimage for a key exists.
    /// @return lengths_ The slot of the mapping that stores the length of the preimage.
    /// @return parts_ The slot of the mapping that stores the preimage parts.
    function _getKeyKindMappingSlots(bytes32 _key)
        public
        pure
        returns (bytes32 partOk_, bytes32 lengths_, bytes32 parts_)
    {
        assembly {
            let ty := shr(248, _key)

            mstore(0x00, ty)
            partOk_ := keccak256(0x00, 0x20)
            mstore(0x00, or(ty, shl(0x08, 0x01)))
            lengths_ := keccak256(0x00, 0x20)
            mstore(0x00, or(ty, shl(0x08, 0x02)))
            parts_ := keccak256(0x00, 0x20)
        }
    }
}
