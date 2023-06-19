// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract PreimageOracle {
    mapping(bytes32 => uint256) public preimageLengths;
    mapping(bytes32 => mapping(uint256 => bytes32)) public preimageParts;
    mapping(bytes32 => mapping(uint256 => bool)) public preimagePartOk;

    function readPreimage(bytes32 key, uint256 offset)
        external
        view
        returns (bytes32 dat, uint256 datLen)
    {
        require(preimagePartOk[key][offset], "preimage must exist");
        datLen = 32;
        uint256 length = preimageLengths[key];
        // add 8 for the length-prefix part
        if (offset + 32 >= length + 8) {
            datLen = length + 8 - offset;
        }
        dat = preimageParts[key][offset];
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
        preimagePartOk[key][partOffset] = true;
        preimageParts[key][partOffset] = part;
        preimageLengths[key] = size;
    }
}
