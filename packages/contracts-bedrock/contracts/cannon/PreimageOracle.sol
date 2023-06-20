// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { IPreimageOracle } from "./interfaces/IPreimageOracle.sol";
import { MissingPreimage, UnauthorizedCaller } from "./lib/CannonErrors.sol";
import { PreimageKey, PreimageOffset, PreimagePart, PreimageLength } from "./lib/CannonTypes.sol";

/// @title PreimageOracle
/// @notice A contract for storing permissioned pre-images.
contract PreimageOracle is Initializable, IPreimageOracle {
    /// @notice The authorized oracle writer.
    address public oracleWriter;

    /// @notice Stores the lengths of pre-images.
    mapping(PreimageKey => PreimageLength) public preimageLengths;

    /// @notice Stores pre-image parts by key and offset.
    mapping(PreimageKey => mapping(PreimageOffset => PreimagePart)) public preimageParts;

    /// @notice Stores whether a pre-image part has been set.
    mapping(PreimageKey => mapping(PreimageOffset => bool)) public preimagePartOk;

    /// @notice Creates a new pre-image oracle.
    /// @param _oracleWriter The authorized oracle writer.
    constructor(address _oracleWriter) {
        initialize(_oracleWriter);
    }

    /// @notice Initializer.
    function initialize(address _oracleWriter) public initializer {
        oracleWriter = _oracleWriter;
    }

    // TODO(CLI-4104):
    // we need to mix-in the ID of the dispute for local-type keys to avoid collisions,
    // and restrict local pre-image insertion to the dispute-managing contract.
    // For now we permit anyone to write any pre-image unchecked, to make testing easy.
    // This method is DANGEROUS. And NOT FOR PRODUCTION.
    function cheat(
        PreimageOffset partOffset,
        PreimageKey key,
        PreimagePart part,
        PreimageLength size
    ) external {
        preimagePartOk[key][partOffset] = true;
        preimageParts[key][partOffset] = part;
        preimageLengths[key] = size;
    }

    /// @inheritdoc IPreimageOracle
    function readPreimage(PreimageKey key, PreimageOffset offset)
        external
        view
        returns (PreimagePart dat, PreimageLength datLen)
    {
        // Validate that the pre-image part exists
        if (!preimagePartOk[key][offset]) {
            revert MissingPreimage(key, offset);
        }

        // Calculate the length of the pre-image data
        datLen = PreimageLength.wrap(32);
        uint256 length = PreimageLength.unwrap(preimageLengths[key]);

        // add 8 for the length-prefix part
        if (PreimageOffset.unwrap(offset) + 32 >= length + 8) {
            datLen = PreimageLength.wrap(length + 8 - PreimageOffset.unwrap(offset));
        }

        // Retrieve the pre-image data
        dat = preimageParts[key][offset];
    }

    /// @inheritdoc IPreimageOracle
    function computePreimageKey(bytes calldata preimage) external pure returns (PreimageKey key) {
        PreimageLength size;
        assembly {
            size := calldataload(0x24)

            // Leave slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80

            // Store size as a big-endian uint64 at the start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)

            // Copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, preimage.offset, size)

            // Compute the pre-image keccak256 hash (aka the pre-image key)
            let h := keccak256(ptr, size)

            // Mask out prefix byte, replace with type 2 byte
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }
    }

    /// @inheritdoc IPreimageOracle
    function loadKeccak256PreimagePart(PreimageOffset partOffset, bytes calldata preimage)
        external
    {
        // Construct the pre-image key
        PreimageLength size;
        PreimageKey key;
        PreimagePart part;

        assembly {
            // Load the size of the pre-image
            // len(sig) + len(partOffset) + len(preimage offset) = 4 + 32 + 32 = 0x44
            size := calldataload(0x44)

            // Revert if part offset >= size + 8 (i.e. parts must be within bounds)
            if iszero(lt(partOffset, add(size, 8))) {
                revert(0, 0)
            }

            // Leave slots 0x40 and 0x60 untouched,
            // and everything after as scratch-memory.
            let ptr := 0x80

            // Store size as a big-endian uint64 at the start of pre-image
            mstore(ptr, shl(192, size))
            ptr := add(ptr, 8)

            // Copy preimage payload into memory so we can hash and read it.
            calldatacopy(ptr, preimage.offset, size)

            // Note that it includes the 8-byte big-endian uint64 length prefix.
            // This will be zero-padded at the end, since memory at end is clean.
            part := mload(add(sub(ptr, 8), partOffset))

            // Compute the pre-image keccak256 hash (aka the pre-image key)
            let h := keccak256(ptr, size)

            // Mask out prefix byte, replace with type 2 byte
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2))
        }

        // Add the pre-image to storage
        preimagePartOk[key][partOffset] = true;
        preimageParts[key][partOffset] = part;
        preimageLengths[key] = size;
    }
}
