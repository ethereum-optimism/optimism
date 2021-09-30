// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

contract TestOOG {
    function runOutOfGas() public {
        bytes32 h;
        for (uint256 i = 0; i < 100000; i++) {
            h = keccak256(abi.encodePacked(h));
        }
    }
}

contract TestOOGInConstructor is TestOOG {
    constructor() {
        runOutOfGas();
    }
}
