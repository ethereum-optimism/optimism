// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.13;

/* import {Test} from "forge-std/Test.sol"; */
import {Counter} from "src/L1/Counter.sol";

contract CounterTest {

    Counter counter;


    function setUp() public {
        counter = new Counter();
        optimismPortal = new OptimismPortal();
    }

    function test_SetNumber(uint256 x) public {
        counter.setNumber(x);
        require(counter.number() == x, "Not equal");
    }
}
