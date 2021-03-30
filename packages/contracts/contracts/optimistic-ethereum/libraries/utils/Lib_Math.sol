// SPDX-License-Identifier: MIT
pragma solidity >0.5.0 <0.8.0;

/**
 * @title Lib_Math
 */
library Lib_Math {

    /**********************
     * Internal Functions *
     **********************/

    /**
     * Calculates the minumum of two numbers.
     * @param _x First number to compare.
     * @param _y Second number to compare.
     * @return Lesser of the two numbers.
     */
    function min(
        uint256 _x,
        uint256 _y
    )
        internal
        pure
        returns (
            uint256
        )
    {
        if (_x < _y) {
            return _x;
        }

        return _y;
    }
}
