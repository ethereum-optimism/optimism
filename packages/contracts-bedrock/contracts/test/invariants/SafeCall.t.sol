// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { SafeCall } from "../../libraries/SafeCall.sol";

contract SafeCall_Succeeds_Invariants is Test {
    SafeCaller_Actor actor;

    function setUp() public {
        // Create a new safe caller actor.
        actor = new SafeCaller_Actor(vm, false);

        // Set the caller to this contract
        targetSender(address(this));

        // Target the safe caller actor.
        targetContract(address(actor));
    }

    /**
     * @custom:invariant If `callWithMinGas` performs a call, then it must always
     * provide at least the specified minimum gas limit to the subcontext.
     *
     * If the check for remaining gas in `SafeCall.callWithMinGas` passes, the
     * subcontext of the call below it must be provided at least `minGas` gas.
     */
    function invariant_callWithMinGas_alwaysForwardsMinGas_succeeds() public {
        assertEq(actor.numCalls(), 0, "no failed calls allowed");
    }

    function performSafeCallMinGas(uint64 minGas) external {
        SafeCall.callWithMinGas(address(0), minGas, 0, hex"");
    }
}

contract SafeCall_Fails_Invariants is Test {
    SafeCaller_Actor actor;

    function setUp() public {
        // Create a new safe caller actor.
        actor = new SafeCaller_Actor(vm, true);

        // Set the caller to this contract
        targetSender(address(this));

        // Target the safe caller actor.
        targetContract(address(actor));
    }

    /**
     * @custom:invariant `callWithMinGas` reverts if there is not enough gas to pass
     * to the subcontext.
     *
     * If there is not enough gas in the callframe to ensure that `callWithMinGas`
     * can provide the specified minimum gas limit to the subcontext of the call,
     * then `callWithMinGas` must revert.
     */
    function invariant_callWithMinGas_neverForwardsMinGas_reverts() public {
        assertEq(actor.numCalls(), 0, "no successful calls allowed");
    }

    function performSafeCallMinGas(uint64 minGas) external {
        SafeCall.callWithMinGas(address(0), minGas, 0, hex"");
    }
}

contract SafeCaller_Actor is StdUtils {
    bool internal immutable FAILS;

    Vm internal vm;
    uint256 public numCalls;

    constructor(Vm _vm, bool _fails) {
        vm = _vm;
        FAILS = _fails;
    }

    function performSafeCallMinGas(uint64 gas, uint64 minGas) external {
        if (FAILS) {
            // Bound the minimum gas amount to [2500, type(uint48).max]
            minGas = uint64(bound(minGas, 2500, type(uint48).max));
            // Bound the gas passed to [minGas, (((minGas + 200) * 64) / 63)]
            gas = uint64(bound(gas, minGas, (((minGas + 200) * 64) / 63)));
        } else {
            // Bound the minimum gas amount to [2500, type(uint48).max]
            minGas = uint64(bound(minGas, 2500, type(uint48).max));
            // Bound the gas passed to [(((minGas + 200) * 64) / 63) + 500, type(uint64).max]
            gas = uint64(bound(gas, (((minGas + 200) * 64) / 63) + 500, type(uint64).max));
        }

        vm.expectCallMinGas(address(0x00), 0, minGas, hex"");
        bool success = SafeCall.call(
            msg.sender,
            gas,
            0,
            abi.encodeWithSelector(0x2ae57a41, minGas)
        );

        if (success && FAILS) numCalls++;
        if (!FAILS && !success) numCalls++;
    }
}
