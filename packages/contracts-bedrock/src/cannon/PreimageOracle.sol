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

    /// @notice Computes and returns the key for a pre-image.
    /// @param _preimage The pre-image.
    /// @return key_ The pre-image key.
    function computePreimageKey(bytes calldata _preimage) external pure returns (bytes32 key_) {
        uint256 size;
        assembly {
            size := calldataload(0x24)

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
}
