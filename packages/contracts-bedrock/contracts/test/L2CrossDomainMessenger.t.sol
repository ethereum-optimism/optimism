// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer, Reverter, CallerCaller } from "./CommonTest.t.sol";

import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { L2CrossDomainMessenger } from "../L2/L2CrossDomainMessenger.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";
import { Types } from "../libraries/Types.sol";

contract L2CrossDomainMessenger_Test is Messenger_Initializer {
    // Receiver address for testing
    address recipient = address(0xabbaacdc);

    function setUp() public override {
        super.setUp();
    }

    function test_L2MessengerPause() external {
        L2Messenger.pause();
        assert(L2Messenger.paused());
    }

    function testCannot_L2MessengerPause() external {
        vm.expectRevert("Ownable: caller is not the owner");
        vm.prank(address(0xABBA));
        L2Messenger.pause();
    }

    function test_L2MessengerMessageVersion() external {
        (, uint16 version) = Encoding.decodeVersionedNonce(L2Messenger.messageNonce());
        assertEq(version, L2Messenger.MESSAGE_VERSION());
    }

    function test_L2MessengerSendMessage() external {
        bytes memory xDomainCallData = Encoding.encodeCrossDomainMessage(
            L2Messenger.messageNonce(),
            alice,
            recipient,
            0,
            100,
            hex"ff"
        );
        vm.expectCall(
            address(messagePasser),
            abi.encodeWithSelector(
                L2ToL1MessagePasser.initiateWithdrawal.selector,
                address(L1Messenger),
                L2Messenger.baseGas(hex"ff", 100),
                xDomainCallData
            )
        );

        // MessagePassed event
        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            messagePasser.nonce(),
            address(L2Messenger),
            address(L1Messenger),
            0,
            L2Messenger.baseGas(hex"ff", 100),
            xDomainCallData,
            Hashing.hashWithdrawal(
                Types.WithdrawalTransaction({
                    nonce: messagePasser.nonce(),
                    sender: address(L2Messenger),
                    target: address(L1Messenger),
                    value: 0,
                    gasLimit: L2Messenger.baseGas(hex"ff", 100),
                    data: xDomainCallData
                })
            )
        );

        vm.prank(alice);
        L2Messenger.sendMessage(recipient, hex"ff", uint32(100));
    }

    function test_L2MessengerTwiceSendMessage() external {
        uint256 nonce = L2Messenger.messageNonce();
        L2Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        L2Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        // the nonce increments for each message sent
        assertEq(nonce + 2, L2Messenger.messageNonce());
    }

    function test_L2MessengerXDomainSenderReverts() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
    }

    function test_L2MessengerRelayMessageV0Fails() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));

        vm.prank(caller);

        vm.expectRevert(
            "CrossDomainMessenger: only version 1 messages are supported after the Bedrock upgrade"
        );
        L2Messenger.relayMessage(
            0, // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );
    }

    function test_L2MessengerRelayMessageSucceeds() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));

        vm.expectCall(target, hex"1111");

        vm.prank(caller);

        vm.expectEmit(true, true, true, true);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        emit RelayedMessage(hash);

        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // the message hash is in the successfulMessages mapping
        assert(L2Messenger.successfulMessages(hash));
        // it is not in the received messages mapping
        assertEq(L2Messenger.receivedMessages(hash), false);
    }

    // relayMessage: should revert if attempting to relay a message sent to an L1 system contract
    function test_L2MessengerRelayMessageToSystemContract() external {
        address target = address(messagePasser);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        bytes memory message = hex"1111";

        vm.prank(caller);
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            message
        );
    }

    // relayMessage: the xDomainMessageSender is reset to the original value
    function test_L2MessengerxDomainMessageSenderResets() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();

        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        vm.prank(caller);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            address(0),
            address(0),
            0,
            0,
            hex""
        );

        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
    }

    // relayMessage: should revert if paused
    function test_L2MessengerRelayShouldRevertIfPaused() external {
        vm.prank(L2Messenger.owner());
        L2Messenger.pause();

        vm.expectRevert("Pausable: paused");
        L2Messenger.relayMessage(0, address(0), address(0), 0, 0, hex"");
    }

    // relayMessage: should send a successful call to the target contract after the first message
    // fails and ETH gets stuck, but the second message succeeds
    function test_L2MessengerRelayMessageFirstStuckSecondSucceeds() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        uint256 value = 100;

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        vm.etch(target, address(new Reverter()).code);
        vm.deal(address(caller), value);
        vm.prank(caller);
        L2Messenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(L2Messenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(L2Messenger.successfulMessages(hash), false);
        assertEq(L2Messenger.receivedMessages(hash), true);

        vm.expectEmit(true, true, true, true);

        emit RelayedMessage(hash);

        vm.etch(target, address(0).code);
        vm.prank(address(sender));
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(L2Messenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(L2Messenger.successfulMessages(hash), true);
        assertEq(L2Messenger.receivedMessages(hash), true);
    }

    // relayMessage: should revert if recipient is trying to reenter
    function test_L1MessengerRelayMessageRevertsOnReentrancy() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        bytes memory message = abi.encodeWithSelector(
            L2Messenger.relayMessage.selector,
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            message
        );

        vm.etch(target, address(new CallerCaller()).code);

        vm.expectEmit(true, true, true, true, target);

        emit WhatHappened(
            false,
            abi.encodeWithSignature("Error(string)", "ReentrancyGuard: reentrant call")
        );

        vm.prank(caller);
        vm.expectCall(target, message);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            0, // value
            0,
            message
        );

        assertEq(L2Messenger.successfulMessages(hash), false);
        assertEq(L2Messenger.receivedMessages(hash), true);
    }
}
