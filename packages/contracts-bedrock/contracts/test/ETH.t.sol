// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { CommonTest } from "./CommonTest.t.sol";
import { ETH } from "../L2/ETH.sol";
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";

contract ETH_Test is CommonTest {
    ETH eth;

    function setUp() external {
        eth = new ETH();
    }

    function test_metadata() external {
        assertEq(eth.name(), "Ether");
        assertEq(eth.symbol(), "ETH");
        assertEq(eth.decimals(), 18);
    }

    function test_crossDomain() external {
        assertEq(eth.l2Bridge(), Lib_PredeployAddresses.L2_STANDARD_BRIDGE);
        assertEq(eth.l1Token(), address(0));
    }

    function test_transfer() external {
        vm.expectRevert("ETH: transfer is disabled");
        eth.transfer(alice, 100);
    }

    function test_approve() external {
        vm.expectRevert("ETH: approve is disabled");
        eth.approve(alice, 100);
    }

    function test_transferFrom() external {
        vm.expectRevert("ETH: transferFrom is disabled");
        eth.transferFrom(bob, alice, 100);
    }

    function test_increaseAllowance() external {
        vm.expectRevert("ETH: increaseAllowance is disabled");
        eth.increaseAllowance(alice, 100);
    }

    function test_decreaseAllowance() external {
        vm.expectRevert("ETH: decreaseAllowance is disabled");
        eth.decreaseAllowance(alice, 100);
    }

    function test_mint() external {
        vm.expectRevert("ETH: mint is disabled");
        eth.mint(alice, 100);
    }

    function test_burn() external {
        vm.expectRevert("ETH: burn is disabled");
        eth.burn(alice, 100);
    }
}
