// SPDX-License-Identifier: MIT

pragma solidity >=0.7.0;

contract ReturnOne {
    function get() external pure returns(uint256) {
        return 1;
    }
}
