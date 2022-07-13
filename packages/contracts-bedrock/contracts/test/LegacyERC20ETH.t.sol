// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { CommonTest } from "./CommonTest.t.sol";
import { LegacyERC20ETH } from "../legacy/LegacyERC20ETH.sol";
import { PredeployAddresses } from "../libraries/PredeployAddresses.sol";

contract LegacyERC20ETH_Test is CommonTest {
    LegacyERC20ETH eth;

    function setUp() external {
        eth = new LegacyERC20ETH();
    }

    function test_metadata() external {
        assertEq(eth.name(), "Ether");
        assertEq(eth.symbol(), "ETH");
        assertEq(eth.decimals(), 18);
    }

    function test_crossDomain() external {
        assertEq(eth.l2Bridge(), PredeployAddresses.L2_STANDARD_BRIDGE);
        assertEq(eth.l1Token(), address(0));
    }

    function test_transfer() external {
        vm.expectRevert("LegacyERC20ETH: transfer is disabled");
        eth.transfer(alice, 100);
    }

    function test_approve() external {
        vm.expectRevert("LegacyERC20ETH: approve is disabled");
        eth.approve(alice, 100);
    }

    function test_transferFrom() external {
        vm.expectRevert("LegacyERC20ETH: transferFrom is disabled");
        eth.transferFrom(bob, alice, 100);
    }

    function test_increaseAllowance() external {
        vm.expectRevert("LegacyERC20ETH: increaseAllowance is disabled");
        eth.increaseAllowance(alice, 100);
    }

    function test_decreaseAllowance() external {
        vm.expectRevert("LegacyERC20ETH: decreaseAllowance is disabled");
        eth.decreaseAllowance(alice, 100);
    }

    function test_mint() external {
        vm.expectRevert("LegacyERC20ETH: mint is disabled");
        eth.mint(alice, 100);
    }

    function test_burn() external {
        vm.expectRevert("LegacyERC20ETH: burn is disabled");
        eth.burn(alice, 100);
    }
}
