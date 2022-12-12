// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { MintManager } from "../governance/MintManager.sol";
import { GovernanceToken } from "../governance/GovernanceToken.sol";

contract MintManager_Test is CommonTest {
    address constant owner = address(0x1234);
    address constant rando = address(0x5678);
    GovernanceToken internal gov;
    MintManager internal manager;

    function setUp() external {
        vm.prank(owner);
        gov = new GovernanceToken();

        vm.prank(owner);
        manager = new MintManager(owner, address(gov));

        vm.prank(owner);
        gov.transferOwnership(address(manager));
    }

    function test_constructor_succeeds() external {
        assertEq(manager.owner(), owner);
        assertEq(address(manager.governanceToken()), address(gov));
    }

    function test_mint_fromOwner_succeeds() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);
    }

    function test_mint_fromNotOwner_reverts() external {
        // Mint from rando fails.
        vm.prank(rando);
        vm.expectRevert("Ownable: caller is not the owner");
        manager.mint(owner, 100);
    }

    function test_mint_afterPeriodElapsed_succeeds() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);

        // Mint again after period elapsed (2% max).
        vm.warp(block.timestamp + manager.MINT_PERIOD() + 1);
        vm.prank(owner);
        manager.mint(owner, 2);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 102);
    }

    function test_mint_beforePeriodElapsed_reverts() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);

        // Mint again.
        vm.prank(owner);
        vm.expectRevert("MintManager: minting not permitted yet");
        manager.mint(owner, 100);

        // Token balance does not increase.
        assertEq(gov.balanceOf(owner), 100);
    }

    function test_mint_moreThanCap_reverts() external {
        // Mint once.
        vm.prank(owner);
        manager.mint(owner, 100);

        // Token balance increases.
        assertEq(gov.balanceOf(owner), 100);

        // Mint again (greater than 2% max).
        vm.warp(block.timestamp + manager.MINT_PERIOD() + 1);
        vm.prank(owner);
        vm.expectRevert("MintManager: mint amount exceeds cap");
        manager.mint(owner, 3);

        // Token balance does not increase.
        assertEq(gov.balanceOf(owner), 100);
    }

    function test_upgrade_fromOwner_succeeds() external {
        // Upgrade to new manager.
        vm.prank(owner);
        manager.upgrade(rando);

        // New manager is rando.
        assertEq(gov.owner(), rando);
    }

    function test_upgrade_fromNotOwner_reverts() external {
        // Upgrade from rando fails.
        vm.prank(rando);
        vm.expectRevert("Ownable: caller is not the owner");
        manager.upgrade(rando);
    }

    function test_upgrade_toZeroAddress_reverts() external {
        // Upgrade to zero address fails.
        vm.prank(owner);
        vm.expectRevert("MintManager: mint manager cannot be the zero address");
        manager.upgrade(address(0));
    }
}
