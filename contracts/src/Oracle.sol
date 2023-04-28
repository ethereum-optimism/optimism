// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;


contract Oracle {

    mapping (bytes32 => uint256) public preimageLengths;
    mapping (bytes32 => mapping(uint256 => bytes32)) preimageParts;
    mapping (bytes32 => mapping(uint256 => bool)) preimagePartOk;

    function readPreimage(bytes32 key, uint256 offset) external view returns (bytes32 dat, uint256 datLen) {
        require(preimagePartOk[key][offset], "preimage must exist");
        datLen = 32;
        uint256 length = preimageLengths[key];
        // TODO: insert length prefix before data
        if(offset + 32 >= length) {
            datLen = length - offset;
        }
        dat = preimageParts[key][offset];
    }

    // TODO: we need to mix-in the ID of the dispute for local-type keys to avoid collisions,
    // and restrict local pre-image insertion to the dispute-managing contract.
    // For now we permit anyone to write any pre-image unchecked, to make testing easy.
    // This method is DANGEROUS. And NOT FOR PRODUCTION.
    function cheat(uint256 partOffset, bytes32 key, bytes32 part, uint256 size) external {
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
            // calldata layout: 4 (sel) + 0x20 (part offset) + 0x20 (start offset) + 0x20 (size) + preimage payload
            let startOffset := calldataload(0x24)
            if not(eq(startOffset, 0x44)) { // must always point to expected location of the size value.
                revert(0, 0)
            }
            size := calldataload(0x44)
            if iszero(lt(partOffset, size)) { // revert if part offset >= size (i.e. parts must be within bounds)
                revert(0, 0)
            }
            let ptr := 0x80 // we leave solidity slots 0x40 and 0x60 untouched, and everything after as scratch-memory.
            calldatacopy(ptr, 0x64, size) // copy preimage payload into memory so we can hash and read it.
            part := mload(add(ptr, partOffset))  // this will be zero-padded at the end, since memory at end is clean.
            let h := keccak256(ptr, size) // compute preimage keccak256 hash
            key := or(and(h, not(shl(248, 0xFF))), shl(248, 2)) // mask out prefix byte, replace with type 2 byte
        }
        preimagePartOk[key][partOffset] = true;
        preimageParts[key][partOffset] = part;
        preimageLengths[key] = size;
    }
}
