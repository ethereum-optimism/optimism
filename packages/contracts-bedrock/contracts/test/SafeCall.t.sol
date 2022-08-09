// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { SafeCall } from "../libraries/SafeCall.sol";

contract SafeCall_Test is CommonTest {
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
        // don't call the console
        vm.assume(
            to != address(0x000000000000000000636F6e736F6c652e6c6f67)
        );
        // don't call the create2 deployer
        vm.assume(
            to != address(0x4e59b44847b379578588920cA78FbF26c0B4956C)
        );
        // don't send funds to self
        vm.assume(from != to);

        assertEq(from.balance, 0, "from balance is 0");
        vm.deal(from, value);
        assertEq(from.balance, value, "from balance not dealt");

        vm.expectCall(
            to,
            value,
            data
        );

        vm.prank(from);
        bool success = SafeCall.call(
            to,
            gas,
            value,
            data
        );

        assertEq(success, true, "call not successful");
        assertEq(to.balance, value, "to balance received");
        assertEq(from.balance, 0, "from balance not drained");
    }
}
