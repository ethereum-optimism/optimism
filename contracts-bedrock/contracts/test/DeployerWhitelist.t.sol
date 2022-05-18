//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { CommonTest } from "./CommonTest.t.sol";
import { DeployerWhitelist } from "../L2/DeployerWhitelist.sol";

contract DeployerWhitelist_Test is CommonTest {
    DeployerWhitelist list;

    function setUp() external {
        list = new DeployerWhitelist();
    }

    // The owner should be address(0)
    function test_owner() external {
        assertEq(list.owner(), address(0));
    }

    // The storage slot for the owner must be the same
    function test_storageSlots() external {
        vm.prank(list.owner());
        list.setOwner(address(1));

        assertEq(
            bytes32(uint256(1)),
            vm.load(address(list), bytes32(uint256(0)))
        );
    }
}
