// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import "src/libraries/DisputeTypes.sol";
import "src/libraries/DisputeErrors.sol";

import { Test } from "forge-std/Test.sol";
import { DisputeGameFactory, IDisputeGameFactory } from "src/dispute/DisputeGameFactory.sol";
import { IDisputeGame } from "src/dispute/interfaces/IDisputeGame.sol";
import { DelayedWETH } from "src/dispute/weth/DelayedWETH.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { CommonTest } from "test/setup/CommonTest.sol";

contract DelayedWETH_Init is CommonTest {
    event Approval(address indexed src, address indexed guy, uint256 wad);
    event Transfer(address indexed src, address indexed dst, uint256 wad);
    event Deposit(address indexed dst, uint256 wad);
    event Withdrawal(address indexed src, uint256 wad);
    event Unwrap(address indexed src, uint256 wad);

    function setUp() public virtual override {
        super.enableFaultProofs();
        super.setUp();

        // Transfer ownership of delayed WETH to the test contract.
        vm.prank(deploy.mustGetAddress("SystemOwnerSafe"));
        delayedWeth.transferOwnership(address(this));
    }
}

contract DelayedWETH_Initialize_Test is DelayedWETH_Init {
    /// @dev Tests that initialization is successful.
    function test_initialize_succeeds() public {
        assertEq(delayedWeth.owner(), address(this));
        assertEq(address(delayedWeth.config()), address(superchainConfig));
    }
}

contract DelayedWETH_Unlock_Test is DelayedWETH_Init {
    /// @dev Tests that unlocking once is successful.
    function test_unlock_once_succeeds() public {
        delayedWeth.unlock(alice, 1 ether);
        (uint256 amount, uint256 timestamp) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount, 1 ether);
        assertEq(timestamp, block.timestamp);
    }

    /// @dev TEsts that unlocking twice is successful and timestamp/amount is updated.
    function test_unlock_twice_succeeds() public {
        // Unlock once.
        uint256 ts = block.timestamp;
        delayedWeth.unlock(alice, 1 ether);
        (uint256 amount1, uint256 timestamp1) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount1, 1 ether);
        assertEq(timestamp1, ts);

        // Go forward in time.
        vm.warp(ts + 1);

        // Unlock again works.
        delayedWeth.unlock(alice, 1 ether);
        (uint256 amount2, uint256 timestamp2) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount2, 2 ether);
        assertEq(timestamp2, ts + 1);
    }
}

contract DelayedWETH_Withdraw_Test is DelayedWETH_Init {
    /// @dev Tests that withdrawing while unlocked and delay has passed is successful.
    function test_withdraw_whileUnlocked_succeeds() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();
        uint256 balance = address(alice).balance;

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, 1 ether);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Withdraw the WETH.
        vm.expectEmit(true, true, false, false);
        emit Withdrawal(address(alice), 1 ether);
        vm.prank(alice);
        delayedWeth.withdraw(alice, 1 ether);
        assertEq(address(alice).balance, balance + 1 ether);
    }

    /// @dev Tests that withdrawing when unlock was not called fails.
    function test_withdraw_whileLocked_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();
        uint256 balance = address(alice).balance;

        // Withdraw fails when unlock not called.
        vm.expectRevert("DelayedWETH: withdrawal not unlocked");
        vm.prank(alice);
        delayedWeth.withdraw(alice, 0 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @dev Tests that withdrawing while locked and delay has not passed fails.
    function test_withdraw_whileLockedNotLongEnough_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();
        uint256 balance = address(alice).balance;

        // Call unlock.
        vm.prank(alice);
        delayedWeth.unlock(alice, 1 ether);

        // Wait for the delay, but not long enough.
        vm.warp(block.timestamp + delayedWeth.delay() - 1);

        // Withdraw fails when delay not met.
        vm.expectRevert("DelayedWETH: withdrawal delay not met");
        vm.prank(alice);
        delayedWeth.withdraw(alice, 1 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @dev Tests that withdrawing more than unlocked amount fails.
    function test_withdraw_tooMuch_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();
        uint256 balance = address(alice).balance;

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, 1 ether);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Withdraw too much fails.
        vm.expectRevert("DelayedWETH: insufficient unlocked withdrawal");
        vm.prank(alice);
        delayedWeth.withdraw(alice, 2 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @dev Tests that withdrawing while paused fails.
    function test_withdraw_whenPaused_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, 1 ether);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Pause the contract.
        address guardian = optimismPortal.GUARDIAN();
        vm.prank(guardian);
        superchainConfig.pause("identifier");

        // Withdraw fails.
        vm.expectRevert("DelayedWETH: contract is paused");
        vm.prank(alice);
        delayedWeth.withdraw(alice, 1 ether);
    }
}

contract DelayedWETH_Recover_Test is DelayedWETH_Init {
    /// @dev Tests that recovering WETH succeeds.
    function test_recover_succeeds() public {
        delayedWeth.transferOwnership(alice);

        // Give the contract some WETH to recover.
        vm.deal(address(delayedWeth), 1 ether);

        // Record the initial balance.
        uint256 initialBalance = address(alice).balance;

        // Recover the WETH.
        vm.prank(alice);
        delayedWeth.recover(1 ether);

        // Verify the WETH was recovered.
        assertEq(address(delayedWeth).balance, 0);
        assertEq(address(alice).balance, initialBalance + 1 ether);
    }

    /// @dev Tests that recovering WETH by non-owner fails.
    function test_recover_byNonOwner_fails() public {
        vm.prank(alice);
        vm.expectRevert("DelayedWETH: not owner");
        delayedWeth.recover(1 ether);
    }

    /// @dev Tests that recovering more than the balance fails.
    function test_recover_moreThanBalance_fails() public {
        vm.deal(address(delayedWeth), 0.5 ether);
        vm.expectRevert("DelayedWETH: insufficient balance");
        delayedWeth.recover(1 ether);
    }
}

contract DelayedWETH_Hold_Test is DelayedWETH_Init {
    /// @dev Tests that holding WETH succeeds.
    function test_hold_succeeds() public {
        uint256 amount = 1 ether;
        vm.expectEmit(true, true, true, false);
        emit Approval(address(this), alice, amount);
        delayedWeth.hold(alice, amount);
        assertEq(delayedWeth.allowance(address(this), alice), amount);
    }

    /// @dev Tests that holding WETH by non-owner fails.
    function test_hold_byNonOwner_fails() public {
        vm.prank(alice);
        vm.expectRevert("DelayedWETH: not owner");
        delayedWeth.hold(bob, 1 ether);
    }
}
