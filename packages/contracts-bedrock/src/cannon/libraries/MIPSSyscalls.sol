// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { MIPSMemory } from "src/cannon/libraries/MIPSMEmory.sol";
import { IPreimageOracle } from "src/cannon/interfaces/IPreimageOracle.sol";
import { PreimageKeyLib } from "src/cannon/PreimageKeyLib.sol";

library MIPSSyscalls {
    uint32 internal constant FD_STDIN = 0;
    uint32 internal constant FD_STDOUT = 1;
    uint32 internal constant FD_STDERR = 2;
    uint32 internal constant FD_HINT_READ = 3;
    uint32 internal constant FD_HINT_WRITE = 4;
    uint32 internal constant FD_PREIMAGE_READ = 5;
    uint32 internal constant FD_PREIMAGE_WRITE = 6;

    uint32 internal constant EBADF = 0x9;
    uint32 internal constant EINVAL = 0x16;

    function getSyscallArgs(uint32[32] memory _registers)
        internal
        pure
        returns (uint32 sysCallNum_, uint32 a0_, uint32 a1_, uint32 a2_)
    {
        sysCallNum_ = _registers[2];

        a0_ = _registers[4];
        a1_ = _registers[5];
        a2_ = _registers[6];

        return (sysCallNum_, a0_, a1_, a2_);
    }

    function handleMmap(
        uint32 _a0,
        uint32 _a1,
        uint32 _heap
    )
        internal
        pure
        returns (uint32 v0_, uint32 v1_, uint32 newHeap_)
    {
        v1_ = uint32(0);
        newHeap_ = _heap;

        uint32 sz = _a1;
        if (sz & 4095 != 0) {
            // adjust size to align with page size
            sz += 4096 - (sz & 4095);
        }
        if (_a0 == 0) {
            v0_ = _heap;
            newHeap_ += sz;
        } else {
            v0_ = _a0;
        }

        return (v0_, v1_, newHeap_);
    }

    function handleSyscallRead(
        uint32 _a0,
        uint32 _a1,
        uint32 _a2,
        bytes32 _preimageKey,
        uint32 _preimageOffset,
        bytes32 _localContext,
        IPreimageOracle _oracle
    )
        internal
        view
        returns (uint32 v0_, uint32 v1_, uint32 newPreimageOffset_)
    {
        v0_ = uint32(0);
        v1_ = uint32(0);
        newPreimageOffset_ = _preimageOffset;

        // args: _a0 = fd, _a1 = addr, _a2 = count
        // returns: v0_ = read, v1_ = err code
        if (_a0 == FD_STDIN) {
            // Leave v0_ and v1_ zero: read nothing, no error
        }
        // pre-image oracle read
        else if (_a0 == FD_PREIMAGE_READ) {
            // verify proof 1 is correct, and get the existing memory.
            uint32 mem = MIPSMemory.readMem(_a1 & 0xFFffFFfc, 1); // mask the addr to align it to 4 bytes
            // If the preimage key is a local key, localize it in the context of the caller.
            if (uint8(_preimageKey[0]) == 1) {
                _preimageKey = PreimageKeyLib.localize(_preimageKey, _localContext);
            }
            (bytes32 dat, uint256 datLen) = _oracle.readPreimage(_preimageKey, _preimageOffset);

            // Transform data for writing to memory
            // We use assembly for more precise ops, and no var count limit
            assembly {
                let alignment := and(_a1, 3) // the read might not start at an aligned address
                let space := sub(4, alignment) // remaining space in memory word
                if lt(space, datLen) { datLen := space } // if less space than data, shorten data
                if lt(_a2, datLen) { datLen := _a2 } // if requested to read less, read less
                dat := shr(sub(256, mul(datLen, 8)), dat) // right-align data
                dat := shl(mul(sub(sub(4, datLen), alignment), 8), dat) // position data to insert into memory
                // word
                let mask := sub(shl(mul(sub(4, alignment), 8), 1), 1) // mask all bytes after start
                let suffixMask := sub(shl(mul(sub(sub(4, alignment), datLen), 8), 1), 1) // mask of all bytes
                // starting from end, maybe none
                mask := and(mask, not(suffixMask)) // reduce mask to just cover the data we insert
                mem := or(and(mem, not(mask)), dat) // clear masked part of original memory, and insert data
            }

            // Write memory back
            MIPSMemory.writeMem(_a1 & 0xFFffFFfc, 1, mem);
            newPreimageOffset_ += uint32(datLen);
            v0_ = uint32(datLen);
        }
        // hint response
        else if (_a0 == FD_HINT_READ) {
            // Don't read into memory, just say we read it all
            // The result is ignored anyway
            v0_ = _a2;
        } else {
            v0_ = 0xFFffFFff;
            v1_ = EBADF;
        }

        return (v0_, v1_, newPreimageOffset_);
    }
}
