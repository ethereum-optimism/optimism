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
    }
}
