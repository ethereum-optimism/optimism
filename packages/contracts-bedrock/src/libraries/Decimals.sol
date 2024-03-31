// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

library Decimals {
    function scale(uint256 _amount, uint8 _decimals, uint8 _target) internal pure returns (uint256) {
        if (_decimals < _target) {
            return _amount * (10 ** (_target - _decimals));
        } else if (_decimals > _target) {
            return _amount / (10 ** (_decimals - _target));
        } else {
            return _amount;
        }
    }
}
