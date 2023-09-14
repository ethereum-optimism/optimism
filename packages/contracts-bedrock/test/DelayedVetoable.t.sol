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
    uint256 operatingDelay = 14 days;
    DelayedVetoable delayedVetoable;
    Reverter reverter;

    function setUp() public override {
        super.setUp();
        delayedVetoable = new DelayedVetoable({
            initiator_: alice,
            vetoer_: bob,
            target_: address(target),
            operatingDelay_: operatingDelay
        });
        // Most tests will use the operating delay, so we call as the initiator with null data
        // to set the delay. For tests that need to use the initial zero delay, we'll modify the
        // value in storage.
        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(hex"");

        reverter = new Reverter();
    }

    /// @dev This function is used to prevent initiating the delay unintentionally..
    /// @param data The data to be used in the call.
    function assumeNonzeroData(bytes memory data) internal pure {
        vm.assume(data.length > 0);
    }
}

contract DelayedVetoable_Getters_Test is DelayedVetoable_Init {
    /// @dev The getters return the expected values when called by the zero address.
    function test_getters() external {
        vm.startPrank(address(0));
        assertEq(delayedVetoable.initiator(), initiator);
        assertEq(delayedVetoable.vetoer(), vetoer);
        assertEq(delayedVetoable.target(), target);
        assertEq(delayedVetoable.delay(), operatingDelay);
    }
}

contract DelayedVetoable_Getters_TestFail is DelayedVetoable_Init {
    /// @dev Check that getter calls from unauthorized entities will revert.
    function test_getters_notZeroAddress_reverts() external {
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
    /// @dev A call can be initiated by the initiator.
    function testFuzz_handleCall_initiation_succeeds(bytes memory data) external {
        assumeNonzeroData(data);

        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Initiated(keccak256(data), data);

        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    /// @dev The delay is inititially set to zero and the call is immediately forwarded.
    function testFuzz_handleCall_initialForwardingImmediately_succeeds(bytes memory data) external {
        assumeNonzeroData(data);

        // Reset the delay to zero
        vm.store(address(delayedVetoable), bytes32(uint256(0)), bytes32(uint256(0)));

        vm.prank(initiator);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        vm.expectCall({ callee: target, data: data });
        emit Forwarded(keccak256(data), data);
        (bool success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    /// @dev The delay can be activated by the vetoer or initiator, and are not forwarded until the delay has passed
    ///      once activated.
    function testFuzz_handleCall_forwardingWithDelay_succeeds(bytes memory data) external {
        assumeNonzeroData(data);

        vm.prank(initiator);
        // it's immediately forwarding for some reason.
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + operatingDelay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        vm.expectCall({ callee: target, data: data });
        (success,) = address(delayedVetoable).call(data);
        assert(success);
    }
}

contract DelayedVetoable_HandleCall_TestFail is DelayedVetoable_Init {
    /// @dev The delay is inititially set to zero and the call is immediately forwarded.
    function test_handleCall_unauthorizedInitiation_reverts() external {
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        (bool success,) = address(delayedVetoable).call(NON_ZERO_DATA);
        assert(success);
    }

    /// @dev The call cannot be forewarded until the delay has passed.
    function testFuzz_handleCall_forwardingTooSoon_reverts(bytes memory data) external {
        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.expectRevert(abi.encodeWithSelector(ForwardingEarly.selector));
        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }

    /// @dev The call cannot be forwarded a second time.
    function testFuzz_handleCall_forwardingTwice_reverts(bytes memory data) external {
        assumeNonzeroData(data);

        // Initiate the call
        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + operatingDelay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        // Forward the call
        vm.expectCall({ callee: target, data: data });
        (success,) = address(delayedVetoable).call(data);
        assert(success);

        // Attempt to foward the same call again.
        vm.expectRevert(abi.encodeWithSelector(Unauthorized.selector, initiator, address(this)));
        (success,) = address(delayedVetoable).call(data);
        assert(success);
    }

    /// @dev If the target reverts, it is bubbled up.
    function testFuzz_handleCall_forwardingTargetReverts_reverts(bytes memory data) external {
        assumeNonzeroData(data);

        vm.etch(target, address(reverter).code);

        vm.prank(initiator);
        (bool success,) = address(delayedVetoable).call(data);

        vm.warp(block.timestamp + operatingDelay);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(keccak256(data), data);

        (success,) = address(delayedVetoable).call(data);
        assertFalse(success);
    }
}
