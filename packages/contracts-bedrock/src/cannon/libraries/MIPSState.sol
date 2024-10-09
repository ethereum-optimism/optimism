// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { InvalidExitedValue } from "src/cannon/libraries/CannonErrors.sol";

library MIPSState {
    struct CpuScalars {
        uint32 pc;
        uint32 nextPC;
        uint32 lo;
        uint32 hi;
    }

    function assertExitedIsValid(uint32 _exited) internal pure {
        if (_exited > 1) {
            revert InvalidExitedValue();
        }
    }
}
