// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

import { Counter } from "./Counter.sol";

contract CounterTest {
    Counter counter;

    function setUp() public {
        counter = new Counter();
    }

    function test_SetNumber(uint256 x) public {
        counter.setNumber(x);
        require(counter.number() == x, "Not equal");
    }
}
