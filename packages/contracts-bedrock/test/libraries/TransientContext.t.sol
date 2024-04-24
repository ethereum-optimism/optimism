// SPDX-License-Identifier: MIT
pragma solidity ^0.8.24;

// Testing utilities
import { Test } from "forge-std/Test.sol";

// Target contractS
import { TransientContext } from "src/libraries/TransientContext.sol";
import { TransientReentrancyAware } from "src/libraries/TransientContext.sol";

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

    function test_increment_overflow_succeeds() public {
        uint256 _startingCallDepth = type(uint256).max;
        assembly {
            tstore(CALL_DEPTH_SLOT, _startingCallDepth)
        }
        assertEq(TransientContext.callDepth(), _startingCallDepth);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), 0);
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

    function test_decrement_underflow_succeeds() public {
        assertEq(TransientContext.callDepth(), 0);

        TransientContext.decrement();

        assertEq(TransientContext.callDepth(), type(uint256).max);
    }

    function testFuzz_get_succeeds(bytes32 _slot, uint256 _value) public {
        assertEq(TransientContext.get(_slot), 0);

        bytes32 tSlot = keccak256(abi.encodePacked(TransientContext.callDepth(), _slot));
        assembly {
            tstore(tSlot, _value)
        }

        assertEq(TransientContext.get(_slot), _value);
    }

    function testFuzz_set_succeeds(bytes32 _slot, uint256 _value) public {
        TransientContext.set(_slot, _value);
        bytes32 tSlot = keccak256(abi.encodePacked(TransientContext.callDepth(), _slot));
        uint256 tValue;
        assembly {
            tValue := tload(tSlot)
        }
        assertEq(tValue, _value);
    }

    function testFuzz_setGet_succeeds(bytes32 _slot, uint256 _value) public {
        testFuzz_set_succeeds(_slot, _value);
        assertEq(TransientContext.get(_slot), _value);
    }

    function testFuzz_setGet_twice_sameDepth_succeeds(bytes32 _slot, uint256 _value1, uint256 _value2) public {
        assertEq(TransientContext.callDepth(), 0);
        testFuzz_set_succeeds(_slot, _value1);
        assertEq(TransientContext.get(_slot), _value1);

        assertEq(TransientContext.callDepth(), 0);
        testFuzz_set_succeeds(_slot, _value2);
        assertEq(TransientContext.get(_slot), _value2);
    }

    function testFuzz_setGet_twice_differentDepth_succeeds(bytes32 _slot, uint256 _value1, uint256 _value2) public {
        assertEq(TransientContext.callDepth(), 0);
        testFuzz_set_succeeds(_slot, _value1);
        assertEq(TransientContext.get(_slot), _value1);

        TransientContext.increment();

        assertEq(TransientContext.callDepth(), 1);
        testFuzz_set_succeeds(_slot, _value2);
        assertEq(TransientContext.get(_slot), _value2);

        TransientContext.decrement();

        assertEq(TransientContext.callDepth(), 0);
        assertEq(TransientContext.get(_slot), _value1);
    }
}

contract TransientReentrancyAwareTest is TransientContextTest, TransientReentrancyAware {
    function mock(bytes32 _slot, uint256 _value) internal reentrantAware {
        TransientContext.set(_slot, _value);
    }

    function mockMultiDepth(bytes32 _slot, uint256 _value1, uint256 _value2) internal reentrantAware {
        TransientContext.set(_slot, _value1);
        mock(_slot, _value2);
    }

    function testFuzz_reentrantAware_succeeds(uint256 _callDepth, bytes32 _slot, uint256 _value) public {
        vm.assume(_callDepth < type(uint256).max);
        assembly {
            tstore(CALL_DEPTH_SLOT, _callDepth)
        }
        assertEq(TransientContext.callDepth(), _callDepth);

        mock(_slot, _value);

        assertEq(TransientContext.get(_slot), 0);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), _callDepth + 1);
        assertEq(TransientContext.get(_slot), _value);
    }

    function testFuzz_reentrantAware__multiDepth_succeeds(
        uint256 _callDepth,
        bytes32 _slot,
        uint256 _value1,
        uint256 _value2
    )
        public
    {
        vm.assume(_callDepth < type(uint256).max - 1);
        assembly {
            tstore(CALL_DEPTH_SLOT, _callDepth)
        }
        assertEq(TransientContext.callDepth(), _callDepth);

        mockMultiDepth(_slot, _value1, _value2);

        assertEq(TransientContext.get(_slot), 0);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), _callDepth + 1);
        assertEq(TransientContext.get(_slot), _value1);

        TransientContext.increment();
        assertEq(TransientContext.callDepth(), _callDepth + 2);
        assertEq(TransientContext.get(_slot), _value2);
    }
}
