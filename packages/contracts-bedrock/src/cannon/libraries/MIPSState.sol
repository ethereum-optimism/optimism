// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { InvalidExitedValue } from "src/cannon/libraries/CannonErrors.sol";

library MIPSState {
    struct CpuScalars {
        uint64 pc;
        uint64 nextPC;
        uint64 lo;
        uint64 hi;
    }

    function assertExitedIsValid(uint32 exited) internal pure {
        if (exited > 1) {
            revert InvalidExitedValue();
        }
    }
}
