// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Comparison {
    function and(bool _a, bool _b) external pure returns (bool) {
        return _a && _b;
    }

    function or(bool _a, bool _b) external pure returns (bool) {
        return _a || _b;
    }

    function lt(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a < _b;
    }

    function gt(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a > _b;
    }

    function eq(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a == _b;
    }

    function ne(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a != _b;
    }

    function lte(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a <= _b;
    }

    function gte(uint256 _a, uint256 _b) external pure returns (bool) {
        return _a >= _b;
    }
}
