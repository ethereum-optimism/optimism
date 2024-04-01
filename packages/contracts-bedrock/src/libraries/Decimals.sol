// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title Decimals
/// @notice Library for scaling uint256 values between different decimal precisions.
library Decimals {
    /// @notice Scales a uint256 value to a number with a different amount of decimals.
    ///         This will underflow or overflow on extreme inputs but is safe on safe
    ///         inputs.
    /// @param _amount   The amount to scale.
    /// @param _decimals The number of decimals the amount currently has.
    /// @param _target   The number of decimals the amount should have.
    function scale(uint256 _amount, uint8 _decimals, uint8 _target) internal pure returns (uint256) {
        if (_decimals == _target) {
            return _amount;
        } else if (_decimals < _target) {
            return _amount * (10 ** (_target - _decimals));
        } else { // (_decimals > _target)
            return _amount / (10 ** (_decimals - _target));
        }
    }
}
