//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { Lib_PredeployAddresses } from "../libraries/Lib_PredeployAddresses.sol";
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

    // withrdraw
    // - requires amount == msg.value
    function test_cannotWithdrawEthWithoutSendingIt() external {
        assertEq(address(messagePasser).balance, 0);

        vm.expectRevert("ETH withdrawals must include sufficient ETH value.");
        vm.prank(alice, alice);
        L2Bridge.withdraw(
            address(Lib_PredeployAddresses.OVM_ETH),
            100,
            1000,
            hex""
        );
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

    function test_withdraw_onlyEOA() external {
        // This contract has 100 L2Token
        deal(address(L2Token), address(this), 100, true);

        vm.expectRevert("Account not EOA");
        L2Bridge.withdraw(
            address(L2Token),
            100,
            1000,
            hex""
        );
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

    // finalizeDeposit
    // - only callable by l1TokenBridge
    // - supported token pair emits DepositFinalized
    // - invalid deposit emits DepositFailed
    // - invalid deposit calls Withdrawer.initiateWithdrawal
    function test_finalizeDeposit_failsToCompleteOutboundTransfer() external {
        // TODO: events and calls
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.otherBridge()))
        );
        address invalidL2Token = address(0x1234);
        vm.prank(address(L2Messenger));
        vm.expectEmit(true, true, true, true);
        emit ERC20BridgeInitiated(
            invalidL2Token,
            address(L1Token),
            alice,
            alice,
            100,
            hex""
        );
        vm.expectEmit(true, true, true, true);
        emit ERC20BridgeFailed(
            invalidL2Token,
            address(L1Token),
            alice,
            alice,
            100,
            hex""
        );
        L2Bridge.finalizeDeposit(
            address(L1Token),
            invalidL2Token,
            alice,
            alice,
            100,
            hex""
        );
    }

    // finalizeBridgeERC20
    // - fails when the local token's address equals bridge address
    function test_ERC20BridgeFailed_whenLocalTokenIsBridge() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.otherBridge()))
        );
        // fails when the local token's address equals bridge address
        vm.expectEmit(true, true, true, true);
        emit ERC20BridgeFailed(address(L2Bridge), address(L1Token), alice, bob, 100, hex"");

        vm.prank(address(L2Messenger));
        L2Bridge.finalizeDeposit(address(L1Token), address(L2Bridge), alice, bob, 100, hex"");
    }
}

