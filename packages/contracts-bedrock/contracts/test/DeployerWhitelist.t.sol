// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { DeployerWhitelist } from "../legacy/DeployerWhitelist.sol";

contract DeployerWhitelist_Test is CommonTest {
    DeployerWhitelist list;

    function setUp() public virtual override {
        list = new DeployerWhitelist();
    }

    // The owner should be address(0)
    function test_owner_succeeds() external {
        assertEq(list.owner(), address(0));
    }

    // The storage slot for the owner must be the same
    function test_storageSlots_succeeds() external {
        vm.prank(list.owner());
        list.setOwner(address(1));

        assertEq(bytes32(uint256(1)), vm.load(address(list), bytes32(uint256(0))));
    }
}
