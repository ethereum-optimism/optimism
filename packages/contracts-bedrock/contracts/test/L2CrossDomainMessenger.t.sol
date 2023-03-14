// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer, Reverter, ConfigurableCaller } from "./CommonTest.t.sol";

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

    function test_messageVersion_succeeds() external {
        (, uint16 version) = Encoding.decodeVersionedNonce(L2Messenger.messageNonce());
        assertEq(version, L2Messenger.MESSAGE_VERSION());
    }

    function test_sendMessage_succeeds() external {
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
            messagePasser.messageNonce(),
            address(L2Messenger),
            address(L1Messenger),
            0,
            L2Messenger.baseGas(hex"ff", 100),
            xDomainCallData,
            Hashing.hashWithdrawal(
                Types.WithdrawalTransaction({
                    nonce: messagePasser.messageNonce(),
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

    function test_sendMessage_twice_succeeds() external {
        uint256 nonce = L2Messenger.messageNonce();
        L2Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        L2Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        // the nonce increments for each message sent
        assertEq(nonce + 2, L2Messenger.messageNonce());
    }

    function test_xDomainSender_senderNotSet_reverts() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
    }

    function test_relayMessage_v2_reverts() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));

        // Expect a revert.
        vm.expectRevert(
            "CrossDomainMessenger: only version 0 or 1 messages are supported at this time"
        );

        // Try to relay a v2 message.
        vm.prank(caller);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 2), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );
    }

    function test_relayMessage_succeeds() external {
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
        assertEq(L2Messenger.failedMessages(hash), false);
    }

    // relayMessage: should revert if attempting to relay a message sent to an L1 system contract
    function test_relayMessage_toSystemContract_reverts() external {
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
    function test_xDomainMessageSender_reset_succeeds() external {
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

    // relayMessage: should send a successful call to the target contract after the first message
    // fails and ETH gets stuck, but the second message succeeds
    function test_relayMessage_retry_succeeds() external {
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
        assertEq(L2Messenger.failedMessages(hash), true);

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
        assertEq(L2Messenger.failedMessages(hash), true);
    }

    // relayMessage: Should revert if the recipient is trying to reenter with the
    // same message.
    function test_relayMessage_reentrancySameMessage_reverts() external {
        ConfigurableCaller caller = new ConfigurableCaller();
        address target = address(caller);
        address sender = address(L1Messenger);
        address l1XDMAlias = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        bytes memory callMessage = abi.encodeWithSelector(caller.call.selector);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            callMessage
        );

        // Act as the L1XDM and call the `relayMessage` function with the `innerMessage`.
        vm.prank(l1XDMAlias);
        vm.expectCall(target, callMessage);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            callMessage
        );

        // Assert that the message failed to be relayed
        assertFalse(L2Messenger.successfulMessages(hash));
        assertTrue(L2Messenger.failedMessages(hash));

        // Set the configurable caller's target to `L2Messenger` and set the payload to `relayMessage(...)`.
        caller.setDoRevert(false);
        caller.setTarget(address(L2Messenger));
        caller.setPayload(
            abi.encodeWithSelector(
                L2Messenger.relayMessage.selector,
                Encoding.encodeVersionedNonce(0, 1),
                sender,
                target,
                0,
                0,
                callMessage
            )
        );

        // Attempt to replay the failed message, which will *not* immediately revert this time around,
        // but attempt to reenter `relayMessage` with the same message hash. The reentrancy attempt should
        // revert.
        vm.expectEmit(true, true, true, true, target);
        emit WhatHappened(
            false,
            abi.encodeWithSignature("Error(string)", "ReentrancyGuard: reentrant call")
        );
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            callMessage
        );

        // Assert that the message still failed to be relayed.
        assertFalse(L2Messenger.successfulMessages(hash));
        assertTrue(L2Messenger.failedMessages(hash));
    }

    // relayMessage: should not revert if the recipient reenters `relayMessage` with a different
    // message hash.
    function test_relayMessage_reentrancyDiffMessage_succeeds() external {
        ConfigurableCaller caller = new ConfigurableCaller();
        address target = address(caller);
        address sender = address(L1Messenger);
        address l1XDMAlias = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));

        bytes memory messageA = abi.encodeWithSelector(caller.call.selector);
        bytes memory messageB = hex"";

        bytes32 hashA = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            messageA
        );
        bytes32 hashB = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            messageB
        );

        // Act as the L1XDM and call the `relayMessage` function with both `messageA` and `messageB`.
        vm.startPrank(l1XDMAlias);

        vm.expectCall(target, messageA);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            messageA
        );
        vm.expectCall(target, messageB);
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            messageB
        );

        // Stop acting as the L1XDM
        vm.stopPrank();

        // Assert that both messages failed to be relayed
        assertFalse(L2Messenger.successfulMessages(hashA));
        assertFalse(L2Messenger.successfulMessages(hashB));
        assertTrue(L2Messenger.failedMessages(hashA));
        assertTrue(L2Messenger.failedMessages(hashB));

        // Set the configurable caller's target to `L2Messenger` and set the payload to `relayMessage(...)`.
        caller.setDoRevert(false);
        caller.setTarget(address(L2Messenger));
        caller.setPayload(
            abi.encodeWithSelector(
                L2Messenger.relayMessage.selector,
                Encoding.encodeVersionedNonce(0, 1),
                sender,
                target,
                0,
                0,
                messageB
            )
        );

        // Attempt to replay the failed message, which will *not* immediately revert this time around,
        // but attempt to reenter `relayMessage` with messageB. The reentrancy attempt should succeed
        // because the message hashes are different.
        vm.expectEmit(true, true, true, true, target);
        emit WhatHappened(true, hex"");
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1),
            sender,
            target,
            0,
            0,
            messageA
        );

        // Assert that both messages are now in the `successfulMessages` mapping.
        assertTrue(L2Messenger.successfulMessages(hashA));
        assertTrue(L2Messenger.successfulMessages(hashB));
    }
}
