// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

contract Assert {
    function t(bool _val) external pure {
        require(_val);
    }

    function f(bool _val) external pure {
        require(!_val);
    }
}
