// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Bridge_Initializer } from "./CommonTest.t.sol";
import { stdStorage, StdStorage } from "forge-std/Test.sol";
import { CrossDomainMessenger } from "../universal/CrossDomainMessenger.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { console } from "forge-std/console.sol";
import { StandardBridge } from "../universal/StandardBridge.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Types } from "../libraries/Types.sol";
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { OptimismMintableERC20 } from "../universal/OptimismMintableERC20.sol";

contract L2StandardBridge_Test is Bridge_Initializer {
    using stdStorage for StdStorage;

    function test_initialize_succeeds() external {
        assertEq(address(L2Bridge.messenger()), address(L2Messenger));
        assertEq(L1Bridge.l2TokenBridge(), address(L2Bridge));
        assertEq(address(L2Bridge.OTHER_BRIDGE()), address(L1Bridge));
    }

    // receive
    // - can accept ETH
    function test_receive_succeeds() external {
        assertEq(address(messagePasser).balance, 0);
        uint256 nonce = L2Messenger.messageNonce();

        bytes memory message = abi.encodeWithSelector(
            StandardBridge.finalizeBridgeETH.selector,
            alice,
            alice,
            100,
            hex""
        );
        uint64 baseGas = L2Messenger.baseGas(message, 200_000);
        bytes memory withdrawalData = abi.encodeWithSelector(
            CrossDomainMessenger.relayMessage.selector,
            nonce,
            address(L2Bridge),
            address(L1Bridge),
            100,
            200_000,
            message
        );
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: address(L2Messenger),
                target: address(L1Messenger),
                value: 100,
                gasLimit: baseGas,
                data: withdrawalData
            })
        );

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeInitiated(alice, alice, 100, hex"");

        // L2ToL1MessagePasser will emit a MessagePassed event
        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            nonce,
            address(L2Messenger),
            address(L1Messenger),
            100,
            baseGas,
            withdrawalData,
            withdrawalHash
        );

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessage(address(L1Bridge), address(L2Bridge), message, nonce, 200_000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessageExtension1(address(L2Bridge), 100);

        vm.expectCall(
            address(L2Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L1Bridge),
                message,
                200_000 // StandardBridge's RECEIVE_DEFAULT_GAS_LIMIT
            )
        );

        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            abi.encodeWithSelector(
                L2ToL1MessagePasser.initiateWithdrawal.selector,
                address(L1Messenger),
                baseGas,
                withdrawalData
            )
        );

        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(address(0), Predeploys.LEGACY_ERC20_ETH, alice, alice, 100, hex"");

        vm.prank(alice, alice);
        (bool success, ) = address(L2Bridge).call{ value: 100 }(hex"");
        assertEq(success, true);
        assertEq(address(messagePasser).balance, 100);
    }

    // withrdraw
    // - requires amount == msg.value
    function test_withdraw_insufficientValue_reverts() external {
        assertEq(address(messagePasser).balance, 0);

        vm.expectRevert("StandardBridge: bridging ETH must include sufficient ETH value");
        vm.prank(alice, alice);
        L2Bridge.withdraw(address(Predeploys.LEGACY_ERC20_ETH), 100, 1000, hex"");
    }

    // withdraw
    // - token is burned
    // - emits WithdrawalInitiated
    // - calls Withdrawer.initiateWithdrawal
    function test_withdraw_withdrawingERC20_succeeds() external {
        // Alice has 100 L2Token
        deal(address(L2Token), alice, 100, true);
        assertEq(L2Token.balanceOf(alice), 100);
        uint256 nonce = L2Messenger.messageNonce();
        bytes memory message = abi.encodeWithSelector(
            StandardBridge.finalizeBridgeERC20.selector,
            address(L1Token),
            address(L2Token),
            alice,
            alice,
            100,
            hex""
        );
        uint64 baseGas = L2Messenger.baseGas(message, 1000);
        bytes memory withdrawalData = abi.encodeWithSelector(
            CrossDomainMessenger.relayMessage.selector,
            nonce,
            address(L2Bridge),
            address(L1Bridge),
            0,
            1000,
            message
        );
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: address(L2Messenger),
                target: address(L1Messenger),
                value: 0,
                gasLimit: baseGas,
                data: withdrawalData
            })
        );

        vm.expectEmit(true, true, true, true);
        emit ERC20BridgeInitiated(address(L2Token), address(L1Token), alice, alice, 100, hex"");

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            nonce,
            address(L2Messenger),
            address(L1Messenger),
            0,
            baseGas,
            withdrawalData,
            withdrawalHash
        );

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessage(address(L1Bridge), address(L2Bridge), message, nonce, 1000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessageExtension1(address(L2Bridge), 0);

        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.expectCall(
            address(L2Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L1Bridge),
                message,
                1000
            )
        );

        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            abi.encodeWithSelector(
                L2ToL1MessagePasser.initiateWithdrawal.selector,
                address(L1Messenger),
                baseGas,
                withdrawalData
            )
        );

        // The L2Bridge should burn the tokens
        vm.expectCall(
            address(L2Token),
            abi.encodeWithSelector(OptimismMintableERC20.burn.selector, alice, 100)
        );

        vm.prank(alice, alice);
        L2Bridge.withdraw(address(L2Token), 100, 1000, hex"");

        assertEq(L2Token.balanceOf(alice), 0);
    }

    function test_withdraw_notEOA_reverts() external {
        // This contract has 100 L2Token
        deal(address(L2Token), address(this), 100, true);

        vm.expectRevert("StandardBridge: function can only be called from an EOA");
        L2Bridge.withdraw(address(L2Token), 100, 1000, hex"");
    }

    // withdrawTo
    // - token is burned
    // - emits WithdrawalInitiated w/ correct recipient
    // - calls Withdrawer.initiateWithdrawal
    function test_withdrawTo_withdrawingERC20_succeeds() external {
        deal(address(L2Token), alice, 100, true);
        assertEq(L2Token.balanceOf(alice), 100);
        uint256 nonce = L2Messenger.messageNonce();
        bytes memory message = abi.encodeWithSelector(
            StandardBridge.finalizeBridgeERC20.selector,
            address(L1Token),
            address(L2Token),
            alice,
            bob,
            100,
            hex""
        );
        uint64 baseGas = L2Messenger.baseGas(message, 1000);
        bytes memory withdrawalData = abi.encodeWithSelector(
            CrossDomainMessenger.relayMessage.selector,
            nonce,
            address(L2Bridge),
            address(L1Bridge),
            0,
            1000,
            message
        );
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: nonce,
                sender: address(L2Messenger),
                target: address(L1Messenger),
                value: 0,
                gasLimit: baseGas,
                data: withdrawalData
            })
        );

        vm.expectEmit(true, true, true, true);
        emit ERC20BridgeInitiated(address(L2Token), address(L1Token), alice, bob, 100, hex"");

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            nonce,
            address(L2Messenger),
            address(L1Messenger),
            0,
            baseGas,
            withdrawalData,
            withdrawalHash
        );

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessage(address(L1Bridge), address(L2Bridge), message, nonce, 1000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(true, true, true, true);
        emit SentMessageExtension1(address(L2Bridge), 0);

        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(address(L1Token), address(L2Token), alice, bob, 100, hex"");

        vm.expectCall(
            address(L2Messenger),
            abi.encodeWithSelector(
                CrossDomainMessenger.sendMessage.selector,
                address(L1Bridge),
                message,
                1000
            )
        );

        vm.expectCall(
            Predeploys.L2_TO_L1_MESSAGE_PASSER,
            abi.encodeWithSelector(
                L2ToL1MessagePasser.initiateWithdrawal.selector,
                address(L1Messenger),
                baseGas,
                withdrawalData
            )
        );

        // The L2Bridge should burn the tokens
        vm.expectCall(
            address(L2Token),
            abi.encodeWithSelector(OptimismMintableERC20.burn.selector, alice, 100)
        );

        vm.prank(alice, alice);
        L2Bridge.withdrawTo(address(L2Token), bob, 100, 1000, hex"");

        assertEq(L2Token.balanceOf(alice), 0);
    }

    // finalizeDeposit
    // - only callable by l1TokenBridge
    // - supported token pair emits DepositFinalized
    // - invalid deposit calls Withdrawer.initiateWithdrawal
    function test_finalizeDeposit_depositingERC20_succeeds() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );

        vm.expectCall(
            address(L2Token),
            abi.encodeWithSelector(OptimismMintableERC20.mint.selector, alice, 100)
        );

        // Should emit both the bedrock and legacy events
        vm.expectEmit(true, true, true, true, address(L2Bridge));
        emit ERC20BridgeFinalized(address(L2Token), address(L1Token), alice, alice, 100, hex"");

        vm.expectEmit(true, true, true, true, address(L2Bridge));
        emit DepositFinalized(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.prank(address(L2Messenger));
        L2Bridge.finalizeDeposit(address(L1Token), address(L2Token), alice, alice, 100, hex"");
    }

    function test_finalizeDeposit_depositingETH_succeeds() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );

        // Should emit both the bedrock and legacy events
        vm.expectEmit(true, true, true, true, address(L2Bridge));
        emit ERC20BridgeFinalized(
            address(L2Token), // localToken
            address(L1Token), // remoteToken
            alice,
            alice,
            100,
            hex""
        );

        vm.expectEmit(true, true, true, true, address(L2Bridge));
        emit DepositFinalized(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.prank(address(L2Messenger));
        L2Bridge.finalizeDeposit(address(L1Token), address(L2Token), alice, alice, 100, hex"");
    }

    function test_finalizeBridgeETH_incorrectValue_reverts() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );
        vm.deal(address(L2Messenger), 100);
        vm.prank(address(L2Messenger));
        vm.expectRevert("StandardBridge: amount sent does not match amount required");
        L2Bridge.finalizeBridgeETH{ value: 50 }(alice, alice, 100, hex"");
    }

    function test_finalizeBridgeETH_sendToSelf_reverts() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );
        vm.deal(address(L2Messenger), 100);
        vm.prank(address(L2Messenger));
        vm.expectRevert("StandardBridge: cannot send to self");
        L2Bridge.finalizeBridgeETH{ value: 100 }(alice, address(L2Bridge), 100, hex"");
    }

    function test_finalizeBridgeETH_sendToMessenger_reverts() external {
        vm.mockCall(
            address(L2Bridge.messenger()),
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );
        vm.deal(address(L2Messenger), 100);
        vm.prank(address(L2Messenger));
        vm.expectRevert("StandardBridge: cannot send to messenger");
        L2Bridge.finalizeBridgeETH{ value: 100 }(alice, address(L2Messenger), 100, hex"");
    }
}

contract L2StandardBridge_FinalizeBridgeETH_Test is Bridge_Initializer {
    function test_finalizeBridgeETH_succeeds() external {
        address messenger = address(L2Bridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeWithSelector(CrossDomainMessenger.xDomainMessageSender.selector),
            abi.encode(address(L2Bridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);

        vm.expectEmit(true, true, true, true);
        emit ETHBridgeFinalized(alice, alice, 100, hex"");

        L2Bridge.finalizeBridgeETH{ value: 100 }(alice, alice, 100, hex"");
    }
}
