// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { FixedPointMathLib } from "@rari-capital/solmate/src/utils/FixedPointMathLib.sol";

/**
 * @title Lib_FeeSmoothing
 */
library Lib_FeeSmoothing {
    /**
     * @notice Weight applied to average
     */
    uint256 constant internal prevWeight = 7e18;

    /**
     * @notice Weight applied to value being added to average
     */
    uint256 constant internal nextWeight = 3e18;

    /**
     * @notice Computes a rolling average
     */
    function rollingAverage(
        uint256 prev,
        uint256 next
    ) internal returns (uint256) {
        uint256 a = saturatingMul(prev, FixedPointMathLib.divWadDown(prevWeight, 1e19));
        uint256 b = saturatingMul(next, FixedPointMathLib.divWadDown(nextWeight, 1e19));
        return saturatingAdd(a, b) / 1e18;
    }

    /**
     * @notice Saturating multiplication
     */
    function saturatingMul(uint256 a, uint256 b) internal returns (uint256) {
        unchecked {
            if (a == 0) {
                return 0;
            }
            uint256 c = a * b;
            if (c / a != b) {
                return type(uint256).max;
            }
            return c;
        }
    }

    /**
     * @notice Saturating addition
     */
    function saturatingAdd(uint256 a, uint256 b) internal returns (uint256) {
        unchecked {
            uint256 c = a + b;
            if (c < a) {
                return type(uint256).max;
            }
            return c;
        }
    }
}
