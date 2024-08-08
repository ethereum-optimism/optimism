// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

library MIPSState {
    struct CpuScalars {
        uint64 pc;
        uint64 nextPC;
        uint64 lo;
        uint64 hi;
    }
}
