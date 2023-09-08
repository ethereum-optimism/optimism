// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest, Reverter } from "./CommonTest.t.sol";
import { DelayedVetoable } from "../src/universal/DelayedVetoable.sol";

contract DelayedVetoable_Init is CommonTest {
    event Initiated(bytes32 indexed callHash, bytes data);
    event Forwarded(bytes32 indexed callHash, bytes data);

    address target = address(0xabba);
    uint256 delay = 14 days;
    DelayedVetoable delayedVetoable;
    Reverter reverter;

    function setUp() public override {
        super.setUp();
        delayedVetoable = new DelayedVetoable({
            target: address(target),
            delay: delay
        });
        reverter = new Reverter();
    }
}

contract DelayedVetoable_HandleCall_Test is DelayedVetoable_Init {
    function testFuzz_handleCall_initiation_succeeds(bytes memory data) external {
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Initiated(keccak256(data), data);

        (bool success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    function testFuzz_handleCall_forwarding_succeeds(bytes memory data) external {
        // Initiate the call
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + delay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        vm.expectCall({ callee: target, data: data });
        (success,) = address(delayedVetoable).call(data);
        assert(success);
    }
}

contract DelayedVetoable_HandleCall_TestFail is DelayedVetoable_Init {
    function test_handleCall_forwardingTooSoon_reverts(bytes memory data) external {
        (bool success,) = address(delayedVetoable).call(data);

        vm.expectRevert();
        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }

    function test_handleCall_forwardingTargetReverts_reverts(bytes memory data) external {
        vm.etch(target, address(reverter).code);

        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + delay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        vm.expectCall({ callee: target, data: data });
        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }
}
