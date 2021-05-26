// SPDX-License-Identifier: MIT
pragma solidity >=0.7.0;

contract TestOOG {
    constructor() {
        bytes32 h;
        for (uint256 i = 0; i < 100000; i++) {
            h = keccak256(abi.encodePacked(h));
        }
    }
}
