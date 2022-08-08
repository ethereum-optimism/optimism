// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { ExcessivelySafeCall } from "excessively-safe-call/src/ExcessivelySafeCall.sol";

import { console } from "forge-std/console.sol";

contract ExcessivelySafeCall_Test is CommonTest {

    function test_safeCall(
        address from,
        address to,
        uint256 gas,
        uint64 value,
        bytes memory data
    ) external {
        vm.assume(from.balance == 0);
        vm.assume(to.balance == 0);
        // no precompiles
        vm.assume(uint160(to) > 10);
        // don't call the vm
        vm.assume(to != address(vm));
        vm.assume(from != address(vm));

        assertEq(from.balance, 0, "from balance is 0");
        vm.expectCall(
            to,
            value,
            data
        );
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");
        vm.prank(from);

        (bool success, ) = ExcessivelySafeCall.excessivelySafeCall(
            to,
            gas,
            value,
            0,
            data
        );

        /*
        (bool success, ) = to.call{gas: gas, value: value}(data);
        */

        assertEq(success, true, "call not successful");
        assertEq(to.balance, value, "to balance received");
        assertEq(from.balance, 0, "from balance not drained");
        console.log(from.balance, to.balance);
    }

}
