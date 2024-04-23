// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { TransientContext } from "src/libraries/TransientContext.sol";

contract TransientContextTest is Test {
    bytes32 internal constant CALL_DEPTH_SLOT = 0x7a74fd168763fd280eaec3bcd2fd62d0e795027adc8183a693c497a7c2b10b5c;

    function testFuzz_callDepth_succeeds(uint256 _value) public {
        assembly {
            tstore(CALL_DEPTH_SLOT, _value)
        }
        assertEq(TransientContext.callDepth(), _value);
    }

    function test_increment_zero_succeeds() public view {
        assertEq(TransientContext.get(CALL_DEPTH_SLOT), 0);
    }

    function test_increment_one_succeeds() public {
        TransientContext.increment();
        assertEq(TransientContext.get(CALL_DEPTH_SLOT), 1);
    }

    function test_increment_many_succeeds(uint256 _times) public {
        for (uint256 i = 0; i < _times; i++) {
            TransientContext.increment();
            assertEq(TransientContext.get(CALL_DEPTH_SLOT), i + 1);
        }
    }

    function test_decrement_zero_succeeds() public {
        assembly {
            tstore(CALL_DEPTH_SLOT, 1)
        }
        assertEq(TransientContext.get(CALL_DEPTH_SLOT), 1);
    }
}

contract TransientReentrancyAwareTest is Test {
    function test_callDepth_succeeds() public view {
        assertEq(TransientContext.callDepth(), 0);
    }
}
