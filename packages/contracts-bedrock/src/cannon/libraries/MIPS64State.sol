// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Libraries
import { InvalidExitedValue } from "src/cannon/libraries/CannonErrors.sol";

library MIPS64State {
    struct CpuScalars {
        uint64 pc;
        uint64 nextPC;
        uint64 lo;
        uint64 hi;
    }

    struct Features {
        bool supportWorkingSysGetRandom;
    }

    function assertExitedIsValid(uint32 _exited) internal pure {
        if (_exited > 1) {
            revert InvalidExitedValue();
        }
    }

    function featuresForVersion(uint256 _version) internal pure returns (Features memory features_) {
        if (_version >= 8) {
            features_.supportWorkingSysGetRandom = true;
        }
    }
}
