// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { StdInvariant } from "forge-std/StdInvariant.sol";
import { Vm } from "forge-std/Vm.sol";
import { DelayedVetoable } from "../../src/universal/DelayedVetoable.sol";

contract DelayedVetoable_Invariant_Harness is StdInvariant, Test {
    event Forwarded(bytes data);

    DelayedVetoable delayedVetoable;

    // The address that delayedVetoable will call to
    address dvTarget;

    function setUp() public {
        delayedVetoable = new DelayedVetoable({
            target: address(dvTarget)
        });

        targetContract(address(delayedVetoable));
    }
}

contract DelayedVetoable_Invariants is DelayedVetoable_Invariant_Harness {
    /// @custom:invariant Calls are always forwarded to the target
    function invariant_callIsForwarded_succeeds() public {
        vm.expectEmit(true, false, false, false, address(delayedVetoable));
        emit Forwarded(hex"");
    }
}
