// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

contract MultiStorage {
    uint256 public slotA;
    uint256 public slotB;
    address public owner;

    constructor() {
        owner = msg.sender;
    }

    function setValues(uint256 _a, uint256 _b) external {
        slotA = _a;
        slotB = _b;
    }
}