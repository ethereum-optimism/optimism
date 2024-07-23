// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

library MIPSState {
    struct CpuScalars {
        uint32 pc;
        uint32 nextPC;
        uint32 lo;
        uint32 hi;
    }
}
