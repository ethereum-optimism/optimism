// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { MIPSState as st } from "src/cannon/libraries/MIPSState.sol";

library MIPSSyscalls {
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
}
