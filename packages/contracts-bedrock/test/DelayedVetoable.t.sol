// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest, Reverter } from "./CommonTest.t.sol";
import { DelayedVetoable } from "../src/universal/DelayedVetoable.sol";

contract DelayedVetoable_Init is CommonTest {
    event Forwarded(bytes data);

    address target = address(0xabba);
    DelayedVetoable delayedVetoable;
    Reverter reverter;

    function setUp() public override {
        super.setUp();
        delayedVetoable = new DelayedVetoable({
            target: address(target)
        });
        reverter = new Reverter();
    }
}

contract DelayedVetoable_HandleCall_Test is DelayedVetoable_Init {
    function testFuzz_handleCall_succeeds(bytes memory data) external {
        vm.expectCall(target, data);
        vm.expectEmit(true, false, false, true, address(delayedVetoable));
        emit Forwarded(data);

        (bool success,) = address(delayedVetoable).call(data);
        assert(success);
    }
}

contract DelayedVetoable_HandleCall_TestFail is DelayedVetoable_Init {
    function test_handleCall_reverts() external {
        vm.expectCall(target, NON_ZERO_DATA);
        vm.expectRevert();
        // including data will call the fallback
        (bool success,) = address(delayedVetoable).call(NON_ZERO_DATA);
        assertFalse(success);
    }
}
