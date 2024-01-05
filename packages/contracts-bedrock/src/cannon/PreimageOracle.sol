// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "./PreimageKeyLib.sol";
import { LibKeccak } from "@lib-keccak/LibKeccak.sol";
import "./libraries/CannonErrors.sol";

/// @title PreimageOracle
/// @notice A contract for storing permissioned pre-images.
contract PreimageOracle is IPreimageOracle {
    /// @notice Metadata related to a large pre-image that is currently being absorbed.
    struct LargePreimageMeta {
        uint128 offset;
        uint64 claimedSize;
        uint64 size;
        bytes32 preimagePart;
    }

    /// @notice Mapping of pre-image keys to pre-image lengths.
    mapping(bytes32 => uint256) public preimageLengths;
    /// @notice Mapping of pre-image keys to pre-image parts.
    mapping(bytes32 => mapping(uint256 => bytes32)) public preimageParts;
    /// @notice Mapping of pre-image keys to pre-image part offsets.
    mapping(bytes32 => mapping(uint256 => bool)) public preimagePartOk;

    /// @notice Mapping of addresses to large pre-image metadata.
    mapping(address => LargePreimageMeta) public largePreimageMeta;
    /// @notice Mapping of addresses to Keccak256 state matrices. Used to submit very large pre-images to the oracle.
    mapping(address => uint64[25]) public stateMatrices;

    /// @inheritdoc IPreimageOracle
    function readPreimage(bytes32 _key, uint256 _offset) external view returns (bytes32 dat_, uint256 datLen_) {
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

    /// @inheritdoc IPreimageOracle
    function loadLocalData(
        uint256 _ident,
        bytes32 _localContext,
        bytes32 _word,
        uint256 _size,
        uint256 _partOffset
    )
        external
        returns (bytes32 key_)
    {
        // Compute the localized key from the given local identifier.
        key_ = PreimageKeyLib.localizeIdent(_ident, _localContext);

        // Revert if the given part offset is not within bounds.
        if (_partOffset > _size + 8 || _size > 32) {
            revert PartOffsetOOB();
        }

        // Prepare the local data part at the given offset
        bytes32 part;
        assembly {
            // Clean the memory in [0x20, 0x40)
            mstore(0x20, 0x00)

            // Store the full local data in scratch space.
            mstore(0x00, shl(192, _size))
            mstore(0x08, _word)

            // Prepare the local data part at the requested offset.
            part := mload(_partOffset)
        }

        // Store the first part with `_partOffset`.
        preimagePartOk[key_][_partOffset] = true;
        preimageParts[key_][_partOffset] = part;
        // Assign the length of the preimage at the localized key.
        preimageLengths[key_] = _size;
    }

    /// @inheritdoc IPreimageOracle
    function loadKeccak256PreimagePart(uint256 _partOffset, bytes calldata _preimage) external {
        uint256 size;
        bytes32 key;
        bytes32 part;
        assembly {
            // len(sig) + len(partOffset) + len(preimage offset) = 4 + 32 + 32 = 0x44
            size := calldataload(0x44)

            // revert if part offset > size+8 (i.e. parts must be within bounds)
            if gt(_partOffset, add(size, 8)) {
                // Store "PartOffsetOOB()"
                mstore(0, 0xfe254987)
                // Revert with "PartOffsetOOB()"
                revert(0x1c, 4)
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

    /// @notice Resets the caller's large pre-image metadata in preparation for beginning the absorption of a new
    ///         large keccak256 pre-image.
    function initLargeKeccak256Preimage(uint128 _offset, uint64 _claimedSize) external {
        largePreimageMeta[msg.sender] =
            LargePreimageMeta({ offset: _offset, claimedSize: _claimedSize, size: 0, preimagePart: bytes32(0) });

        LibKeccak.StateMatrix memory state;
        stateMatrices[msg.sender] = state.state;
    }

    /// @notice Absorbs a part of the caller's large keccak256 pre-image.
    function absorbLargePreimagePart(bytes calldata _data, bool _finalize) external {
        // Revert if the input length is not a multiple of the block size and we're not finalizing the absorbtion.
        bool isModBlockSize = _data.length % LibKeccak.BLOCK_SIZE_BYTES == 0;
        if (!(isModBlockSize || _finalize)) revert InvalidInputLength();

        // If we're finalizing the absorbtion, pad the final blocks of input data passed. Otherwise, we've been passed
        // full block(s) as asserted above, and we copy the data into memory as-is.
        // MAYBE: Just make `LibKeccak` accept a `bytes memory` rather than a `bytes calldata`?
        bytes memory data;
        if (_finalize) {
            data = LibKeccak.pad(_data);
        } else {
            data = _data;
        }

        uint256 dataPtr;
        assembly {
            dataPtr := add(data, 0x20)
        }

        // Pull the state into memory for the absorbtion.
        LibKeccak.StateMatrix memory state = LibKeccak.StateMatrix(stateMatrices[msg.sender]);

        // Grab the number of bytes that have already been absorbed.
        LargePreimageMeta storage preimageMeta = largePreimageMeta[msg.sender];
        uint256 currentSize = preimageMeta.size;
        uint256 offset = preimageMeta.offset;

        if (offset < 8 && currentSize == 0) {
            // In the case that the offset is less than 8, we need to assign the preimage part in a special way. The
            // first 8 bytes of the preimage data stored in the oracle is the length of the preimage in the form of a
            // big-endian uint64.
            uint64 claimedSize = preimageMeta.claimedSize;
            bytes32 preimagePart;
            assembly {
                mstore(0x00, shl(192, claimedSize))
                mstore(0x08, mload(dataPtr))
                preimagePart := mload(offset)
            }
            preimageMeta.preimagePart = preimagePart;
        } else if (offset >= currentSize && offset < currentSize + _data.length) {
            // If the preimage part is in the data we're about to absorb, persist the part to the caller's large
            // preimaage metadata.
            bytes32 preimagePart;
            assembly {
                preimagePart := mload(add(dataPtr, sub(offset, sub(currentSize, 1))))
            }
            preimageMeta.preimagePart = preimagePart;
        }

        // Absorb the data into the sponge.
        bytes memory blockBuffer = new bytes(136);
        for (uint256 i; i < _data.length; i += LibKeccak.BLOCK_SIZE_BYTES) {
            // Pull the current block into the processing buffer.
            assembly {
                let blockPtr := add(dataPtr, i)
                mstore(add(blockBuffer, 0x20), mload(blockPtr))
                mstore(add(blockBuffer, 0x40), mload(add(blockPtr, 0x20)))
                mstore(add(blockBuffer, 0x60), mload(add(blockPtr, 0x40)))
                mstore(add(blockBuffer, 0x80), mload(add(blockPtr, 0x60)))
                mstore(add(blockBuffer, 0xA0), and(mload(add(blockPtr, 0x80)), shl(192, 0xFFFFFFFFFFFFFFFF)))
            }

            LibKeccak.absorb(state, blockBuffer);
            LibKeccak.permutation(state);
        }

        // Update the state and metadata.
        stateMatrices[msg.sender] = state.state;
        largePreimageMeta[msg.sender].size += uint64(_data.length);
    }

    /// @notice Squeezes the caller's large keccak256 pre-image and persists the part into storage.
    function squeezeLargePreimagePart() external {
        // Pull the state into memory for squeezing
        LibKeccak.StateMatrix memory state = LibKeccak.StateMatrix(stateMatrices[msg.sender]);

        // Grab the large preimage metadata.
        LargePreimageMeta memory meta = largePreimageMeta[msg.sender];

        // Revert if the part offset is out of bounds.
        if (meta.offset > meta.size + 8) revert PartOffsetOOB();

        // Revert if the final size is not equal to the claimed size.
        if (meta.size != meta.claimedSize) revert InvalidClaimedSize();

        // Squeeze the data out of the sponge.
        bytes32 finalDigest = LibKeccak.squeeze(state);

        // Compute the preimage key
        bytes32 key;
        assembly {
            key := or(and(finalDigest, not(shl(248, 0xFF))), shl(248, 2))
        }

        // Store the final preimage part.
        preimagePartOk[key][meta.offset] = true;
        preimageParts[key][meta.offset] = meta.preimagePart;
        preimageLengths[key] = meta.size;
    }
}
