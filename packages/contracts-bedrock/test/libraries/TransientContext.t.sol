// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { TransientContext } from "src/libraries/TransientContext.sol";

contract TransientContextTest is Test {
    function test_increment_zero_succeeds(bytes32 _slot) public view {
        assertEq(TransientContext.get(_slot), 0);
    }

    function test_increment_one_succeeds(bytes32 _slot) public view {
        TransientContext.increment();
        assertEq(TransientContext.get(_slot), 1);
    }

    function test_increment_many_succeeds(uint256 _times) public view {
        for (uint256 i = 0; i < _times; i++) {
            TransientContext.increment();
            assertEq(TransientContext.get(_slot), i + 1);
        }
    }
}

contract TransientReentrancyAwareTest is Test {
    function test_callDepth_succeeds() public view {
        assertEq(TransientContext.callDepth(), 0);
    }
}
