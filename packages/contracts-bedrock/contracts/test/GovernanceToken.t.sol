// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "./CommonTest.t.sol";
import { GovernanceToken } from "../governance/GovernanceToken.sol";

contract GovernanceToken_Test is CommonTest {
    address constant owner = address(0x1234);
    address constant rando = address(0x5678);
    GovernanceToken internal gov;

    function setUp() public virtual override {
        super.setUp();
        vm.prank(owner);
        gov = new GovernanceToken();
    }

    function test_constructor_succeeds() external {
        assertEq(gov.owner(), owner);
        assertEq(gov.name(), "Optimism");
        assertEq(gov.symbol(), "OP");
        assertEq(gov.decimals(), 18);
        assertEq(gov.totalSupply(), 0);
    }

    function test_mint_fromOwner_succeeds() external {
        // Mint 100 tokens.
        vm.prank(owner);
        gov.mint(owner, 100);

        // Balances have updated correctly.
        assertEq(gov.balanceOf(owner), 100);
        assertEq(gov.totalSupply(), 100);
    }

    function test_mint_fromNotOwner_reverts() external {
        // Mint 100 tokens as rando.
        vm.prank(rando);
        vm.expectRevert("Ownable: caller is not the owner");
        gov.mint(owner, 100);

        // Balance does not update.
        assertEq(gov.balanceOf(owner), 0);
        assertEq(gov.totalSupply(), 0);
    }

    function test_burn_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando burns their tokens.
        vm.prank(rando);
        gov.burn(50);

        // Balances have updated correctly.
        assertEq(gov.balanceOf(rando), 50);
        assertEq(gov.totalSupply(), 50);
    }

    function test_burnFrom_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando approves owner to burn 50 tokens.
        vm.prank(rando);
        gov.approve(owner, 50);

        // Owner burns 50 tokens from rando.
        vm.prank(owner);
        gov.burnFrom(rando, 50);

        // Balances have updated correctly.
        assertEq(gov.balanceOf(rando), 50);
        assertEq(gov.totalSupply(), 50);
    }

    function test_transfer_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando transfers 50 tokens to owner.
        vm.prank(rando);
        gov.transfer(owner, 50);

        // Balances have updated correctly.
        assertEq(gov.balanceOf(owner), 50);
        assertEq(gov.balanceOf(rando), 50);
        assertEq(gov.totalSupply(), 100);
    }

    function test_approve_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando approves owner to spend 50 tokens.
        vm.prank(rando);
        gov.approve(owner, 50);

        // Allowances have updated.
        assertEq(gov.allowance(rando, owner), 50);
    }

    function test_transferFrom_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando approves owner to spend 50 tokens.
        vm.prank(rando);
        gov.approve(owner, 50);

        // Owner transfers 50 tokens from rando to owner.
        vm.prank(owner);
        gov.transferFrom(rando, owner, 50);

        // Balances have updated correctly.
        assertEq(gov.balanceOf(owner), 50);
        assertEq(gov.balanceOf(rando), 50);
        assertEq(gov.totalSupply(), 100);
    }

    function test_increaseAllowance_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando approves owner to spend 50 tokens.
        vm.prank(rando);
        gov.approve(owner, 50);

        // Rando increases allowance by 50 tokens.
        vm.prank(rando);
        gov.increaseAllowance(owner, 50);

        // Allowances have updated.
        assertEq(gov.allowance(rando, owner), 100);
    }

    function test_decreaseAllowance_succeeds() external {
        // Mint 100 tokens to rando.
        vm.prank(owner);
        gov.mint(rando, 100);

        // Rando approves owner to spend 100 tokens.
        vm.prank(rando);
        gov.approve(owner, 100);

        // Rando decreases allowance by 50 tokens.
        vm.prank(rando);
        gov.decreaseAllowance(owner, 50);

        // Allowances have updated.
        assertEq(gov.allowance(rando, owner), 50);
    }
}
