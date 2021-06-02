// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/**
 * @title Lib_MathUtils
 */
library Lib_MathUtils {

    /**********************
     * Internal Functions *
     **********************/

    function min(
        uint256 _a,
        uint256 _b
    )
        internal
        pure
        returns(
            uint256
        )
    {
        if (_a < _b) {
            return _a;
        }
        return _b;
    }
}
