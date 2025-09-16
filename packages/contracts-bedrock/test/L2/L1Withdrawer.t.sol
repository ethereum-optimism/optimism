// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { IL1Withdrawer } from "interfaces/L2/IL1Withdrawer.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @title L1Withdrawer_Test
/// @notice Tests all functionality of L1Withdrawer including receive, withdrawal, and setters.
contract L1Withdrawer_Test is CommonTest {
    uint256 minWithdrawalAmount = 10 ether;
    uint96 withdrawalGasLimit = 300_000;

    event WithdrawalInitiated(address indexed recipient, uint256 amount);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event MinWithdrawalAmountUpdated(uint256 oldMinWithdrawalAmount, uint256 newMinWithdrawalAmount);
    event RecipientUpdated(address oldRecipient, address newRecipient);
    event WithdrawalGasLimitUpdated(uint96 oldWithdrawalGasLimit, uint96 newWithdrawalGasLimit);

    function setUp() public override {
        // Enable revenue sharing before calling parent setUp
        super.enableRevenueShare();
        super.setUp();
    }

    function testFuzz_receive_belowThreshold_succeeds(uint256 _amount) external {
        _amount = bound(_amount, 0, minWithdrawalAmount - 1);

        vm.deal(address(this), _amount);

        vm.expectEmit(address(l1Withdrawer));
        emit FundsReceived(address(this), _amount, _amount);

        (bool success,) = address(l1Withdrawer).call{ value: _amount }("");

        assertTrue(success);
        assertEq(address(l1Withdrawer).balance, _amount);
        assertEq(address(Predeploys.L2_TO_L1_MESSAGE_PASSER).balance, 0);
    }

    function testFuzz_receive_atOrAboveThreshold_succeeds(uint256 _sendAmount) external {
        _sendAmount = bound(_sendAmount, minWithdrawalAmount, type(uint256).max);

        vm.deal(address(this), _sendAmount);

        vm.expectEmit(address(l1Withdrawer));
        emit FundsReceived(address(this), _sendAmount, _sendAmount);

        vm.expectEmit(address(l1Withdrawer));
        emit WithdrawalInitiated(l1FeesDepositor, _sendAmount);

        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            _sendAmount,
            abi.encodeCall(IL2ToL1MessagePasser.initiateWithdrawal, (l1FeesDepositor, withdrawalGasLimit, hex""))
        );

        (bool success,) = address(l1Withdrawer).call{ value: _sendAmount }("");

        assertTrue(success);
        assertEq(address(l1Withdrawer).balance, 0);
        assertEq(address(Predeploys.L2_TO_L1_MESSAGE_PASSER).balance, _sendAmount);
    }

    function testFuzz_receive_multipleDeposits_succeeds(uint256 _firstAmount, uint256 _secondAmount) external {
        // First amount should not exceed minWithdrawalAmount (so it doesn't trigger withdrawal)
        _firstAmount = bound(_firstAmount, 0, minWithdrawalAmount - 1);

        // Second amount should ensure total reaches threshold to trigger withdrawal
        _secondAmount = bound(_secondAmount, minWithdrawalAmount - _firstAmount, type(uint256).max - _firstAmount);

        uint256 totalAmount = _firstAmount + _secondAmount;

        // First deposit (should not trigger withdrawal)
        vm.deal(address(this), _firstAmount);

        vm.expectEmit(address(l1Withdrawer));
        emit FundsReceived(address(this), _firstAmount, _firstAmount);

        (bool success1,) = address(l1Withdrawer).call{ value: _firstAmount }("");
        assertTrue(success1);
        assertEq(address(l1Withdrawer).balance, _firstAmount);
        assertEq(address(Predeploys.L2_TO_L1_MESSAGE_PASSER).balance, 0);

        // Second deposit (will trigger withdrawal since total >= minWithdrawalAmount)
        vm.deal(address(this), _secondAmount);

        vm.expectEmit(address(l1Withdrawer));
        emit FundsReceived(address(this), _secondAmount, totalAmount);

        vm.expectEmit(address(l1Withdrawer));
        emit WithdrawalInitiated(l1FeesDepositor, totalAmount);

        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            totalAmount,
            abi.encodeCall(IL2ToL1MessagePasser.initiateWithdrawal, (l1FeesDepositor, withdrawalGasLimit, hex""))
        );

        (bool success2,) = address(l1Withdrawer).call{ value: _secondAmount }("");
        assertTrue(success2);

        // Verify withdrawal occurred
        assertEq(address(l1Withdrawer).balance, 0);
        assertEq(address(Predeploys.L2_TO_L1_MESSAGE_PASSER).balance, totalAmount);
    }

    function testFuzz_setMinWithdrawalAmount_asOwner_succeeds(uint256 _newMinWithdrawalAmount) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(l1Withdrawer));
        emit MinWithdrawalAmountUpdated(minWithdrawalAmount, _newMinWithdrawalAmount);

        vm.prank(owner);
        l1Withdrawer.setMinWithdrawalAmount(_newMinWithdrawalAmount);

        assertEq(l1Withdrawer.minWithdrawalAmount(), _newMinWithdrawalAmount);
    }

    function testFuzz_setMinWithdrawalAmount_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        uint256 newMinWithdrawalAmount = 2 ether;

        vm.expectRevert(IL1Withdrawer.L1Withdrawer_OnlyProxyAdminOwner.selector);
        vm.prank(_caller);
        l1Withdrawer.setMinWithdrawalAmount(newMinWithdrawalAmount);

        assertEq(l1Withdrawer.minWithdrawalAmount(), minWithdrawalAmount);
    }

    function testFuzz_setRecipient_asOwner_succeeds(address _newRecipient) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(l1Withdrawer));
        emit RecipientUpdated(l1FeesDepositor, _newRecipient);

        vm.prank(owner);
        l1Withdrawer.setRecipient(_newRecipient);

        assertEq(l1Withdrawer.recipient(), _newRecipient);
    }

    function testFuzz_setRecipient_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        address newRecipient = makeAddr("newRecipient");

        vm.expectRevert(IL1Withdrawer.L1Withdrawer_OnlyProxyAdminOwner.selector);
        vm.prank(_caller);
        l1Withdrawer.setRecipient(newRecipient);

        assertEq(l1Withdrawer.recipient(), l1FeesDepositor);
    }

    function testFuzz_setWithdrawalGasLimit_asOwner_succeeds(uint96 _newWithdrawalGasLimit) external {
        address owner = proxyAdmin.owner();

        vm.expectEmit(address(l1Withdrawer));
        emit WithdrawalGasLimitUpdated(withdrawalGasLimit, _newWithdrawalGasLimit);

        vm.prank(owner);
        l1Withdrawer.setWithdrawalGasLimit(_newWithdrawalGasLimit);

        assertEq(l1Withdrawer.withdrawalGasLimit(), _newWithdrawalGasLimit);
    }

    function testFuzz_setWithdrawalGasLimit_asNonOwner_reverts(address _caller) external {
        address owner = proxyAdmin.owner();
        vm.assume(_caller != owner);

        uint96 newWithdrawalGasLimit = 200_000;

        vm.expectRevert(IL1Withdrawer.L1Withdrawer_OnlyProxyAdminOwner.selector);
        vm.prank(_caller);
        l1Withdrawer.setWithdrawalGasLimit(newWithdrawalGasLimit);

        assertEq(l1Withdrawer.withdrawalGasLimit(), withdrawalGasLimit);
    }
}
