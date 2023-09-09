// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest, Reverter } from "./CommonTest.t.sol";
import { DelayedVetoable } from "../src/L1/DelayedVetoable.sol";

contract DelayedVetoable_Init is CommonTest {
    error Unauthorized(address expected, address actual);
    error ForwardingEarly();

    event Initiated(bytes32 indexed callHash, bytes data);
    event Forwarded(bytes32 indexed callHash, bytes data);
    event Vetoed(bytes32 indexed callHash, bytes data);

    address target = address(0xabba);
    address initiator = alice;
    address vetoer = bob;
    uint256 delay = 14 days;
    DelayedVetoable delayedVetoable;
    Reverter reverter;

    function setUp() public override {
        super.setUp();
        delayedVetoable = new DelayedVetoable({
            initiator_: alice,
            vetoer_: bob,
            target_: address(target),
            delay_: delay
        });
        reverter = new Reverter();
    }
}

contract DelayedVetoable_Getters_Test is DelayedVetoable_Init {
    function test_getters() external {
        vm.startPrank(address(0));
        assertEq(delayedVetoable.initiator(), initiator);
        assertEq(delayedVetoable.vetoer(), vetoer);
        assertEq(delayedVetoable.target(), target);
        assertEq(delayedVetoable.delay(), delay);
    }
}

contract DelayedVetoable_Getters_TestFail is DelayedVetoable_Init {
    function test_getters_notVetoer() external {
        // getter calls from addresses other than the vetoer or zero address will revert in the
        // initiation branch of the proxy.
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        delayedVetoable.initiator();
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        delayedVetoable.vetoer();
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        delayedVetoable.target();
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        delayedVetoable.delay();
    }
}

contract DelayedVetoable_HandleCall_Test is DelayedVetoable_Init {
    function testFuzz_handleCall_initiation_succeeds(bytes memory data) external {
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Initiated(keccak256(data), data);

        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    function testFuzz_handleCall_forwarding_succeeds(bytes memory data) external {
        // Initiate the call
        vm.prank(initiator);
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
    function test_handleCall_unauthorizedInitiation_reverts() external {
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        (bool success,) = address(delayedVetoable).call(hex"");
        assert(success);
    }

    function test_handleCall_forwardingTooSoon_reverts(bytes memory data) external {
        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.expectRevert(abi.encodeWithSelector(ForwardingEarly.selector));
        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }

    function test_handleCall_forwardingTwice_reverts(bytes memory data) external {
        // Initiate the call
        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + delay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        vm.expectCall({ callee: target, data: data });
        (success,) = address(delayedVetoable).call(data);
        assert(success);

        // Attempt to foward the same call again.
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        (success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    function test_handleCall_forwardingTargetReverts_reverts(bytes memory data) external {
        vm.etch(target, address(reverter).code);

        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + delay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }
}

contract DelayedVetoable_Veto_Test is DelayedVetoable_Init {
    function test_veto_succeeds(bytes memory data) external {
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Vetoed(keccak256(data), data);

        vm.prank(vetoer);
        delayedVetoable.veto(data);
    }
}

contract DelayedVetoable_Veto_TestFail is DelayedVetoable_Init {
    function test_veto_notVetoer_reverts() external {
        vm.expectRevert();
        delayedVetoable.veto(hex"");
    }
}
