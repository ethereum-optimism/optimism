// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract SelfDestruction {
    bytes32 public data = 0x0000000000000000000000000000000000000000000000000000000061626364;

    function setData(bytes32 _data) public {
        data = _data;
    }

    function destruct() public {
        address payable self = payable(address(this));
        selfdestruct(self);
    }
}
