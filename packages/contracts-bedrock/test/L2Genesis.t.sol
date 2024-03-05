// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L2Genesis } from "scripts/L2Genesis.s.sol";

contract L2GenesisTest is Test {
    L2Genesis genesis;

    function setUp() public {
        genesis = new L2Genesis();
        genesis.setUp();
    }

    function testFoo() external {
        assertTrue(true);
    }
}
