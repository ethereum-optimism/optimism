// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";
import { Burn } from "src/libraries/Burn.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import "src/dispute/lib/Types.sol";
import "src/dispute/lib/Errors.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";

/// @title DelayedWETH_FallbackGasUser_Harness
/// @notice Contract that burns gas in the fallback function.
contract DelayedWETH_FallbackGasUser_Harness {
    /// @notice Amount of gas to use in the fallback function.
    uint256 public gas;

    /// @param _gas Amount of gas to use in the fallback function.
    constructor(uint256 _gas) {
        gas = _gas;
    }

    /// @notice Burn gas on fallback;
    fallback() external payable {
        Burn.gas(gas);
    }

    /// @notice Burn gas on receive.
    receive() external payable {
        Burn.gas(gas);
    }
}

/// @title DelayedWETH_FallbackReverter_Harness
/// @notice Contract that reverts in the fallback function.
contract DelayedWETH_FallbackReverter_Harness {
    /// @notice Revert on fallback.
    fallback() external payable {
        revert("FallbackReverter: revert");
    }

    /// @notice Revert on receive.
    receive() external payable {
        revert("FallbackReverter: revert");
    }
}

/// @title DelayedWETH_TestInit
/// @notice Reusable test initialization for `DelayedWETH` tests.
abstract contract DelayedWETH_TestInit is CommonTest {
    event Approval(address indexed src, address indexed guy, uint256 wad);
    event Transfer(address indexed src, address indexed dst, uint256 wad);
    event Deposit(address indexed dst, uint256 wad);
    event Withdrawal(address indexed src, uint256 wad);
    event Unwrap(address indexed src, uint256 wad);

    function setUp() public virtual override {
        super.setUp();
    }
}

/// @title DelayedWETH_Initialize_Test
/// @notice Tests the `initialize` function of the `DelayedWETH` contract.
contract DelayedWETH_Initialize_Test is DelayedWETH_TestInit {
    /// @notice Tests that initialization is successful.
    function test_initialize_succeeds() public view {
        assertEq(delayedWeth.proxyAdminOwner(), proxyAdminOwner);
        assertEq(address(delayedWeth.systemConfig()), address(systemConfig));
        assertEq(address(delayedWeth.config()), address(systemConfig.superchainConfig()));
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal initialization
    ///         but confirms that the initValue is not incremented incorrectly if an upgrade
    ///         function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("DelayedWETH", "_initialized");

        // Get the initializer value.
        bytes32 slotVal = vm.load(address(delayedWeth), bytes32(slot.slot));
        uint8 val = uint8(uint256(slotVal) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, delayedWeth.initVersion());
    }

    /// @notice Tests that initialization reverts if called by a non-proxy admin or proxy admin
    ///         owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(delayedWeth.proxyAdmin()) && _sender != delayedWeth.proxyAdminOwner());

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("DelayedWETH", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(delayedWeth), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector.
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender.
        vm.prank(_sender);
        delayedWeth.initialize(ISystemConfig(address(1234)));
    }
}

/// @title DelayedWETH_Unlock_Test
/// @notice Tests the `unlock` function of the `DelayedWETH` contract.
contract DelayedWETH_Unlock_Test is DelayedWETH_TestInit {
    /// @notice Tests that unlocking once is successful.
    /// @param _wad Amount of WETH to unlock.
    function testFuzz_unlock_once_succeeds(uint256 _wad) public {
        delayedWeth.unlock(alice, _wad);
        (uint256 amount, uint256 timestamp) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount, _wad);
        assertEq(timestamp, block.timestamp);
    }

    /// @notice Tests that unlocking twice is successful
    ///         and timestamp/amount is updated.
    /// @param _wad1 First unlock amount.
    /// @param _wad2 Second unlock amount.
    /// @param _timeDelta Time between unlocks.
    function testFuzz_unlock_twice_succeeds(uint256 _wad1, uint256 _wad2, uint256 _timeDelta) public {
        // Bound to prevent overflow on addition.
        _wad1 = bound(_wad1, 0, type(uint128).max);
        _wad2 = bound(_wad2, 0, type(uint128).max);
        _timeDelta = bound(_timeDelta, 1, type(uint128).max);

        // Unlock once.
        uint256 ts = block.timestamp;
        delayedWeth.unlock(alice, _wad1);
        (uint256 amount1, uint256 timestamp1) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount1, _wad1);
        assertEq(timestamp1, ts);

        // Go forward in time.
        vm.warp(ts + _timeDelta);

        // Unlock again works.
        delayedWeth.unlock(alice, _wad2);
        (uint256 amount2, uint256 timestamp2) = delayedWeth.withdrawals(address(this), alice);
        assertEq(amount2, _wad1 + _wad2);
        assertEq(timestamp2, ts + _timeDelta);
    }
}

/// @title DelayedWETH_Withdraw_Test
/// @notice Tests the `withdraw` function of the `DelayedWETH` contract.
contract DelayedWETH_Withdraw_Test is DelayedWETH_TestInit {
    /// @notice Tests that withdrawing while unlocked and
    ///         delay has passed is successful.
    /// @param _wad Amount of WETH to withdraw.
    function testFuzz_withdraw_whileUnlocked_succeeds(uint256 _wad) public {
        _wad = bound(_wad, 0, type(uint192).max);

        // Deposit some WETH.
        vm.deal(alice, _wad);
        vm.prank(alice);
        delayedWeth.deposit{ value: _wad }();
        uint256 balance = address(alice).balance;

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, _wad);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Withdraw the WETH.
        vm.expectEmit(true, true, false, false);
        emit Withdrawal(address(alice), _wad);
        vm.prank(alice);
        delayedWeth.withdraw(_wad);
        assertEq(address(alice).balance, balance + _wad);
    }

    /// @notice Tests that withdrawing when unlock was not called fails.
    function test_withdraw_whileLocked_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();
        uint256 balance = address(alice).balance;

        // Withdraw fails when unlock not called.
        vm.expectRevert("DelayedWETH: withdrawal not unlocked");
        vm.prank(alice);
        delayedWeth.withdraw(0 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @notice Tests that withdrawing while locked and delay has not passed fails.
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
        delayedWeth.withdraw(1 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @notice Tests that withdrawing more than unlocked amount fails.
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
        delayedWeth.withdraw(2 ether);
        assertEq(address(alice).balance, balance);
    }

    /// @notice Tests that withdrawing while paused fails.
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
        address guardian = optimismPortal2.guardian();
        vm.prank(guardian);
        superchainConfig.pause(address(0));

        // Withdraw fails.
        vm.expectRevert("DelayedWETH: contract is paused");
        vm.prank(alice);
        delayedWeth.withdraw(1 ether);
    }

    /// @notice Tests that withdrawing with sub-account
    ///         while unlocked and delay has passed succeeds.
    /// @param _wad Amount of WETH to withdraw.
    function testFuzz_withdraw_withdrawFromWhileUnlocked_succeeds(uint256 _wad) public {
        _wad = bound(_wad, 0, type(uint192).max);

        // Deposit some WETH.
        vm.deal(alice, _wad);
        vm.prank(alice);
        delayedWeth.deposit{ value: _wad }();
        uint256 balance = address(alice).balance;

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, _wad);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Withdraw the WETH.
        vm.expectEmit(true, true, false, false);
        emit Withdrawal(address(alice), _wad);
        vm.prank(alice);
        delayedWeth.withdraw(alice, _wad);
        assertEq(address(alice).balance, balance + _wad);
    }

    /// @notice Tests that withdrawing when unlock was not called fails.
    function test_withdraw_withdrawFromWhileLocked_fails() public {
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

    /// @notice Tests that withdrawing while locked and delay has not passed fails.
    function test_withdraw_withdrawFromWhileLockedNotLongEnough_fails() public {
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

    /// @notice Tests that withdrawing more than unlocked amount fails.
    function test_withdraw_withdrawFromTooMuch_fails() public {
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

    /// @notice Tests that withdrawing while paused fails.
    function test_withdraw_withdrawFromWhenPaused_fails() public {
        // Deposit some WETH.
        vm.prank(alice);
        delayedWeth.deposit{ value: 1 ether }();

        // Unlock the withdrawal.
        vm.prank(alice);
        delayedWeth.unlock(alice, 1 ether);

        // Wait for the delay.
        vm.warp(block.timestamp + delayedWeth.delay() + 1);

        // Pause the contract.
        address guardian = optimismPortal2.guardian();
        vm.prank(guardian);
        superchainConfig.pause(address(0));

        // Withdraw fails.
        vm.expectRevert("DelayedWETH: contract is paused");
        vm.prank(alice);
        delayedWeth.withdraw(alice, 1 ether);
    }
}

/// @title DelayedWETH_Recover_Test
/// @notice Tests the `recover` function of the `DelayedWETH` contract.
contract DelayedWETH_Recover_Test is DelayedWETH_TestInit {
    /// @notice Tests that recovering WETH succeeds. Makes sure that doing so succeeds with any
    ///         amount of ETH in the contract and any amount of gas used in the fallback function
    ///         up to a maximum of 20,000,000 gas. Owner contract should never be using that much
    ///         gas but we might as well set a very large upper bound for ourselves.
    /// @param _amount Amount of WETH to recover.
    /// @param _fallbackGasUsage Amount of gas to use in the fallback function.
    function testFuzz_recover_succeeds(uint256 _amount, uint256 _fallbackGasUsage) public {
        // Assume
        _fallbackGasUsage = bound(_fallbackGasUsage, 0, 20000000);

        // Set up the gas burner.
        DelayedWETH_FallbackGasUser_Harness gasUser = new DelayedWETH_FallbackGasUser_Harness(_fallbackGasUsage);

        // Mock owner to return the gas user.
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(address(gasUser)));

        // Give the contract some WETH to recover.
        vm.deal(address(delayedWeth), _amount);

        // Record the initial balance.
        uint256 initialBalance = address(gasUser).balance;

        // Recover the WETH.
        vm.prank(address(gasUser));
        delayedWeth.recover(_amount);

        // Verify the WETH was recovered.
        assertEq(address(delayedWeth).balance, 0);
        assertEq(address(gasUser).balance, initialBalance + _amount);
    }

    /// @notice Tests that recovering WETH by non-owner fails.
    /// @param _sender Random address for access control.
    function testFuzz_recover_byNonOwner_fails(address _sender) public {
        vm.assume(_sender != proxyAdminOwner);

        // Recover fails.
        vm.expectRevert("DelayedWETH: not owner");
        vm.prank(_sender);
        delayedWeth.recover(1 ether);
    }

    /// @notice Tests that recovering more than the balance
    ///         recovers what it can.
    /// @param _balance Contract balance.
    /// @param _extra Extra amount above balance.
    function testFuzz_recover_moreThanBalance_succeeds(uint256 _balance, uint256 _extra) public {
        _balance = bound(_balance, 0, type(uint128).max);
        _extra = bound(_extra, 1, type(uint128).max);
        uint256 wad = _balance + _extra;

        // Mock owner to return alice.
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(alice));

        // Give the contract some ETH to recover.
        vm.deal(address(delayedWeth), _balance);

        // Record the initial balance.
        uint256 initialBalance = address(alice).balance;

        // Recover the WETH.
        vm.prank(alice);
        delayedWeth.recover(wad);

        // Verify capped at actual balance.
        assertEq(address(delayedWeth).balance, 0);
        assertEq(address(alice).balance, initialBalance + _balance);
    }

    /// @notice Tests that recovering less than the balance
    ///         sends the exact requested amount.
    /// @param _balance Contract balance.
    /// @param _wad Amount to recover (less than balance).
    function testFuzz_recover_partialAmount_succeeds(uint256 _balance, uint256 _wad) public {
        _balance = bound(_balance, 1, type(uint128).max);
        _wad = bound(_wad, 0, _balance - 1);

        // Mock owner to return alice.
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(alice));

        // Give the contract some ETH to recover.
        vm.deal(address(delayedWeth), _balance);

        // Record the initial balance.
        uint256 initialBalance = address(alice).balance;

        // Recover partial amount.
        vm.prank(alice);
        delayedWeth.recover(_wad);

        // Verify exact amount was recovered.
        assertEq(address(delayedWeth).balance, _balance - _wad);
        assertEq(address(alice).balance, initialBalance + _wad);
    }

    /// @notice Tests that recover reverts when recipient reverts.
    function test_recover_whenRecipientReverts_fails() public {
        // Set up the reverter.
        DelayedWETH_FallbackReverter_Harness reverter = new DelayedWETH_FallbackReverter_Harness();

        // Mock owner to return the reverter.
        vm.mockCall(address(proxyAdmin), abi.encodeCall(IProxyAdmin.owner, ()), abi.encode(address(reverter)));

        // Give the contract some WETH to recover.
        vm.deal(address(delayedWeth), 1 ether);

        // Recover fails.
        vm.expectRevert("DelayedWETH: recover failed");
        vm.prank(address(reverter));
        delayedWeth.recover(1 ether);
    }
}

/// @title DelayedWETH_Hold_Test
/// @notice Tests the `hold` function of the `DelayedWETH` contract.
contract DelayedWETH_Hold_Test is DelayedWETH_TestInit {
    /// @notice Tests that holding WETH succeeds.
    /// @param _wad Amount of WETH to hold.
    function testFuzz_hold_byOwner_succeeds(uint256 _wad) public {
        _wad = bound(_wad, 0, type(uint192).max);

        // Pretend to be alice and deposit some WETH.
        vm.deal(alice, _wad);
        vm.prank(alice);
        delayedWeth.deposit{ value: _wad }();

        // Get our balance before.
        uint256 initialBalance = delayedWeth.balanceOf(address(proxyAdminOwner));

        // Hold some WETH.
        vm.expectEmit(true, true, true, false);
        emit Approval(alice, address(proxyAdminOwner), _wad);
        vm.prank(proxyAdminOwner);
        delayedWeth.hold(alice, _wad);

        // Get our balance after.
        uint256 finalBalance = delayedWeth.balanceOf(address(proxyAdminOwner));

        // Verify the transfer.
        assertEq(finalBalance, initialBalance + _wad);
    }

    /// @notice Tests that holding all WETH without
    ///         specifying amount succeeds.
    /// @param _wad Amount of WETH to deposit and hold.
    function testFuzz_hold_withoutAmount_succeeds(uint256 _wad) public {
        _wad = bound(_wad, 0, type(uint192).max);

        // Pretend to be alice and deposit some WETH.
        vm.deal(alice, _wad);
        vm.prank(alice);
        delayedWeth.deposit{ value: _wad }();

        // Get our balance before.
        uint256 initialBalance = delayedWeth.balanceOf(address(proxyAdminOwner));

        // Hold all WETH.
        vm.expectEmit(true, true, true, false);
        emit Approval(alice, address(proxyAdminOwner), _wad);
        vm.prank(proxyAdminOwner);
        delayedWeth.hold(alice); // without amount parameter

        // Get our balance after.
        uint256 finalBalance = delayedWeth.balanceOf(address(proxyAdminOwner));

        // Verify the transfer.
        assertEq(finalBalance, initialBalance + _wad);
    }

    /// @notice Tests that holding WETH by non-owner fails.
    /// @param _sender Random address for access control.
    function testFuzz_hold_byNonOwner_fails(address _sender) public {
        vm.assume(_sender != proxyAdminOwner);

        // Hold fails.
        vm.expectRevert("DelayedWETH: not owner");
        vm.prank(_sender);
        delayedWeth.hold(bob, 1 ether);
    }

    /// @notice Tests that holding all WETH by non-owner
    ///         using the single-arg overload fails.
    /// @param _sender Random address for access control.
    function testFuzz_hold_noAmountNonOwner_fails(address _sender) public {
        vm.assume(_sender != proxyAdminOwner);

        // Hold fails.
        vm.expectRevert("DelayedWETH: not owner");
        vm.prank(_sender);
        delayedWeth.hold(bob);
    }
}

/// @title DelayedWETH_Version_Test
/// @notice Tests the `version` function of the
///         `DelayedWETH` contract.
contract DelayedWETH_Version_Test is DelayedWETH_TestInit {
    /// @notice Tests that the version string is valid semver.
    function test_version_validFormat_succeeds() external view {
        SemverComp.parse(delayedWeth.version());
    }
}
