// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

/// @title PreimageOracle
/// @notice A contract for storing permissioned pre-images.
contract PreimageOracle {
    /// @notice Mapping of pre-image keys to pre-image lengths.
    mapping(bytes32 => uint256) public preimageLengths;
    /// @notice Mapping of pre-image keys to pre-image parts.
    mapping(bytes32 => mapping(uint256 => bytes32)) public preimageParts;
    /// @notice Mapping of pre-image keys to pre-image part offsets.
    mapping(bytes32 => mapping(uint256 => bool)) public preimagePartOk;

    /// @notice Reads a pre-image from the oracle.
    /// @param _key The key of the pre-image to read.
    /// @param _offset The offset of the pre-image to read.
    /// @return dat_ The pre-image data.
    /// @return datLen_ The length of the pre-image data.
    function readPreimage(bytes32 _key, uint256 _offset)
        external
        view
        returns (bytes32 dat_, uint256 datLen_)
    {
        require(preimagePartOk[_key][_offset], "pre-image must exist");

        // Calculate the length of the pre-image data
        // Add 8 for the length-prefix part
        datLen_ = 32;
        uint256 length = preimageLengths[_key];
        if (_offset + 32 >= length + 8) {
            datLen_ = length + 8 - _offset;
        }

        // Retrieve the pre-image data
        dat_ = preimageParts[_key][_offset];
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
        preimagePartOk[key][partOffset] = true;
        preimageParts[key][partOffset] = part;
        preimageLengths[key] = size;
    }

    /// @notice Loads local data into the pre-image oracle in the context of the caller.
    /// @param _bootInfo The boot info struct encoded as a tuple of .
    function loadLocalData(bytes memory _bootInfo) external {
        (
            bytes32 l1head,
            bytes32 l2head,
            bytes32 l2claim,
            uint64 l2ClaimBlockNumber,
            bytes memory l2ChainConfig,
            bytes memory rollupConfig
        ) = abi.decode(_bootInfo, (bytes32, bytes32, bytes32, uint64, bytes, bytes));

        assembly {
            /// Store a value in a mapping
            function storeInMapping(k, v, mappingSlot) {
                // Value slot: `keccak256(k . mappingSlot)`
                mstore(0x00, k)
                mstore(0x20, mappingSlot)
                sstore(keccak256(0x00, 0x40), v)
            }

            /// Store a value in a nested mapping
            function storeInNestedMapping(ka, kb, v, mappingSlot) {
                // Compute the slot of the nested mapping
                mstore(0x00, ka)
                mstore(0x20, mappingSlot)
                let nestedSlot := keccak256(0x00, 0x40)
                // Compute the slot of the value & store it
                mstore(0x00, kb)
                mstore(0x20, nestedSlot)
                sstore(keccak256(0x00, 0x40), v)
            }

            /// Compute the context-specifc key for a given local data identifier
            function contextKey(ident) -> key {
                // Store the global key (1 << 248 | ident)
                mstore(0, or(shl(248, 1), ident))
                // Store the caller to add context to the local data's global key
                mstore(0x20, caller())
                // Hash the data to get the context-specific key
                // localize(k) = H(k .. sender) & ~(0xFF << 248) | (1 << 248)
                key := or(and(keccak256(0, 0x40), not(shl(248, 0xFF))), shl(248, 1))
            }

            /// Store a fixed-size piece of local data
            function storeFixed(ident, offset, size, data) {
                // Grab the context key for the given `ident`
                let k := contextKey(ident)

                // Store the fixed data
                storeInNestedMapping(k, offset, true, preimagePartOk.slot)
                storeInNestedMapping(k, offset, data, preimageParts.slot)
                storeInMapping(k, size, preimageLengths.slot)
            }

            /// Store a dynamic-size piece of local data
            function storeDyn(ident, dataOffset) {
                // Grab the length of the data
                let size := mload(dataOffset)
                // Grab the context key for the given `ident`
                let k := contextKey(ident)

                // Store each component of the preimage key.
                let dataStart := add(dataOffset, 0x20)
                for { let i := 0 } lt(i, size) { i := add(i, 0x20) } {
                    // Load the part at the given offset
                    // TODO(clabby): Verify size.
                    let part := mload(add(dataStart, i))
                    storeInNestedMapping(k, i, true, preimagePartOk.slot)
                    storeInNestedMapping(k, i, part, preimageParts.slot)
                    storeInMapping(k, size, preimageLengths.slot)
                }
            }

            // Store all components of the boot info.
            storeFixed(0, 0, 32, l1head)
            storeFixed(1, 0, 32, l2head)
            storeFixed(2, 0, 32, l2claim)
            storeFixed(3, 0, 32, l2ClaimBlockNumber)
            storeDyn(4, l2ChainConfig)
            storeDyn(5, rollupConfig)
        }
    }

    /// @notice Prepares a pre-image to be read by keccak256 key, starting at
    ///         the given offset and up to 32 bytes (clipped at pre-image length, if out of data).
    /// @param _partOffset The offset of the pre-image to read.
    /// @param _preimage The preimage data.
    function loadKeccak256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external {
        uint256 size;
        bytes32 key;
        bytes32 part;
        assembly {
            // len(sig) + len(partOffset) + len(preimage offset) = 4 + 32 + 32 = 0x44
            size := calldataload(0x44)

            // revert if part offset > size+8 (i.e. parts must be within bounds)
            if gt(_partOffset, add(size, 8)) {
                revert(0, 0)
            }
            // we leave solidity slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80
            // put size as big-endian uint64 at start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)
            // copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, _preimage.offset, size)
            // Note that it includes the 8-byte big-endian uint64 length prefix.
            // this will be zero-padded at the end, since memory at end is clean.
            part := mload(add(sub(ptr, 8), _partOffset))
            let h := keccak256(ptr, size) // compute preimage keccak256 hash
            // mask out prefix byte, replace with type 2 byte
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }
        preimagePartOk[key][_partOffset] = true;
        preimageParts[key][_partOffset] = part;
        preimageLengths[key] = size;
    }

    /// @notice Computes and returns the key for a global keccak pre-image.
    /// @param _preimage The pre-image.
    /// @return key_ The pre-image key.
    function computeKeccak256PreimageKey(bytes calldata _preimage) external pure returns (bytes32 key_) {
        assembly {
            let size := calldataload(0x24)

            // Leave slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80

            // Store size as a big-endian uint64 at the start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)

            // Copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, _preimage.offset, size)

            // Compute the pre-image keccak256 hash (aka the pre-image key)
            let h := keccak256(ptr, size)

            // Mask out prefix byte, replace with type 2 byte
            key_ := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }
    }
}
