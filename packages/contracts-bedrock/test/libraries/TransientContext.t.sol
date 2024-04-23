// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contract
import { TransientContext } from "src/libraries/TransientContext.sol";

contract TransientContextTest is Test {
    bytes32 internal constant CALL_DEPTH_SLOT = 0x7a74fd168763fd280eaec3bcd2fd62d0e795027adc8183a693c497a7c2b10b5c;

    function testFuzz_callDepth_succeeds(uint256 _callDepth) public {
        assembly {
            tstore(CALL_DEPTH_SLOT, _callDepth)
        }

        assertEq(TransientContext.callDepth(), _callDepth);
    }

    function testFuzz_increment_succeeds(uint256 _startingCallDepth) public {
        vm.assume(_startingCallDepth < type(uint256).max);
        assembly {
            tstore(CALL_DEPTH_SLOT, _startingCallDepth)
        }
        assertEq(TransientContext.callDepth(), _startingCallDepth);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), _startingCallDepth + 1);
    }

    function testFuzz_increment_twice_succeeds(uint256 _startingCallDepth) public {
        vm.assume(_startingCallDepth < type(uint256).max - 1);
        testFuzz_increment_succeeds(_startingCallDepth);
        assertEq(TransientContext.callDepth(), _startingCallDepth + 1);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), _startingCallDepth + 2);
    }

    function testFuzz_decrement_succeeds(uint256 _startingCallDepth) public {
        vm.assume(_startingCallDepth > 0);
        assembly {
            tstore(CALL_DEPTH_SLOT, _startingCallDepth)
        }
        assertEq(TransientContext.callDepth(), _startingCallDepth);

        TransientContext.decrement();
        assertEq(TransientContext.callDepth(), _startingCallDepth - 1);
    }

    function testFuzz_decrement_twice_succeeds(uint256 _startingCallDepth) public {
        vm.assume(_startingCallDepth > 1);
        assembly {
            tstore(CALL_DEPTH_SLOT, _startingCallDepth)
        }
        assertEq(TransientContext.callDepth(), _startingCallDepth);

        TransientContext.decrement();
        assertEq(TransientContext.callDepth(), _startingCallDepth - 1);

        testFuzz_decrement_succeeds(_startingCallDepth - 1);
    }

    function test_decrement_fromZero_reverts() public {
        assertEq(TransientContext.callDepth(), 0);

        vm.expectRevert();
        TransientContext.decrement();
    }
}

contract TransientReentrancyAwareTest is Test {
    function test_callDepth_succeeds() public view {
        assertEq(TransientContext.callDepth(), 0);
    }
}
