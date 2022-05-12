//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { console } from "forge-std/console.sol";

contract L2StandardBridge_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

    function setUp() public override {
        super.setUp();
    }

    function test_initialize() external {
        assertEq(
            address(L2Bridge.messenger()),
            address(L2Messenger)
        );

        assertEq(
            address(L2Bridge.otherBridge()),
            address(L1Bridge)
        );
    }

    // receive
    // - can accept ETH
    function test_receive() external {
        assertEq(address(messagePasser).balance, 0);

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeInitiated(alice, alice, 100, hex"");

        // TODO: L2Messenger should be called
        // TODO: L2ToL1MessagePasser should be called
        // TODO: withdrawal hash should be computed correctly
        // TODO: events from each contract

        vm.prank(alice, alice);
        address(L2Bridge).call{ value: 100 }(hex"");
        assertEq(address(messagePasser).balance, 100);
    }

    // withdraw
    // - token is burned
    // - emits WithdrawalInitiated
    // - calls Withdrawer.initiateWithdrawal
    function test_withdraw() external {
        // Alice has 100 L2Token
        deal(address(L2Token), alice, 100, true);
        assertEq(L2Token.balanceOf(alice), 100);

        vm.prank(alice, alice);
        L2Bridge.withdraw(
            address(L2Token),
            100,
            1000,
            hex""
        );

        // TODO: events and calls

        assertEq(L2Token.balanceOf(alice), 0);
    }

    // withdrawTo
    // - token is burned
    // - emits WithdrawalInitiated w/ correct recipient
    // - calls Withdrawer.initiateWithdrawal
    function test_withdrawTo() external {
        deal(address(L2Token), alice, 100, true);

        vm.prank(alice, alice);
        L2Bridge.withdrawTo(
            address(L2Token),
            bob,
            100,
            1000,
            hex""
        );

        // TODO: events and calls

        assertEq(L2Token.balanceOf(alice), 0);
    }

    // finalizeDeposit
    // - only callable by l1TokenBridge
    // - supported token pair emits DepositFinalized
    // - invalid deposit emits DepositFailed
    // - invalid deposit calls Withdrawer.initiateWithdrawal
    function test_finalizeDeposit() external {
        // TODO: events and calls

        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.otherBridge()))
        );
        vm.prank(address(L2Messenger));
        L2Bridge.finalizeDeposit(
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );
    }
}

