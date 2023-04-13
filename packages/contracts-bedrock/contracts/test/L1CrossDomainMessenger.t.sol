// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

/* Testing utilities */
import { Messenger_Initializer, Reverter, ConfigurableCaller } from "./CommonTest.t.sol";
import { L2OutputOracle_Initializer } from "./L2OutputOracle.t.sol";

/* Libraries */
import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { Predeploys } from "../libraries/Predeploys.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

/* Target contract dependencies */
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";

/* Target contract */
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";

contract L1CrossDomainMessenger_Test is Messenger_Initializer {
    // Receiver address for testing
    address recipient = address(0xabbaacdc);

    // Storage slot of the l2Sender
    uint256 constant senderSlotIndex = 50;

    // the version is encoded in the nonce
    function test_messageVersion_succeeds() external {
        (, uint16 version) = Encoding.decodeVersionedNonce(L1Messenger.messageNonce());
        assertEq(version, L1Messenger.MESSAGE_VERSION());
    }

    // sendMessage: should be able to send a single message
    // TODO: this same test needs to be done with the legacy message type
    // by setting the message version to 0
    function test_sendMessage_succeeds() external {
        // deposit transaction on the optimism portal should be called
        vm.expectCall(
            address(op),
            abi.encodeWithSelector(
                OptimismPortal.depositTransaction.selector,
                Predeploys.L2_CROSS_DOMAIN_MESSENGER,
                0,
                L1Messenger.baseGas(hex"ff", 100),
                false,
                Encoding.encodeCrossDomainMessage(
                    L1Messenger.messageNonce(),
                    alice,
                    recipient,
                    0,
                    100,
                    hex"ff"
                )
            )
        );

        // TransactionDeposited event
        vm.expectEmit(true, true, true, true);
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger)),
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            0,
            0,
            L1Messenger.baseGas(hex"ff", 100),
            false,
            Encoding.encodeCrossDomainMessage(
                L1Messenger.messageNonce(),
                alice,
                recipient,
                0,
                100,
                hex"ff"
            )
        );

        // SentMessage event
        vm.expectEmit(true, true, true, true);
        emit SentMessage(recipient, alice, hex"ff", L1Messenger.messageNonce(), 100);

        // SentMessageExtension1 event
        vm.expectEmit(true, true, true, true);
        emit SentMessageExtension1(alice, 0);

        vm.prank(alice);
        L1Messenger.sendMessage(recipient, hex"ff", uint32(100));
    }

    // sendMessage: should be able to send the same message twice
    function test_sendMessage_twice_succeeds() external {
        uint256 nonce = L1Messenger.messageNonce();
        L1Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        L1Messenger.sendMessage(recipient, hex"aa", uint32(500_000));
        // the nonce increments for each message sent
        assertEq(nonce + 2, L1Messenger.messageNonce());
    }

    function test_xDomainSender_notSet_reverts() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L1Messenger.xDomainMessageSender();
    }

    function test_relayMessage_v2_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Expect a revert.
        vm.expectRevert(
            "CrossDomainMessenger: only version 0 or 1 messages are supported at this time"
        );

        // Try to relay a v2 message.
        vm.prank(address(op));
        L2Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 2 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );
    }

    // relayMessage: should send a successful call to the target contract
    function test_relayMessage_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        vm.expectCall(target, hex"1111");

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(op));

        vm.expectEmit(true, true, true, true);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        emit RelayedMessage(hash);

        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // the message hash is in the successfulMessages mapping
        assert(L1Messenger.successfulMessages(hash));
        // it is not in the received messages mapping
        assertEq(L1Messenger.failedMessages(hash), false);
    }

    // relayMessage: should revert if attempting to relay a message sent to an L1 system contract
    function test_relayMessage_toSystemContract_reverts() external {
        // set the target to be the OptimismPortal
        address target = address(op);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.prank(address(op));
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            message
        );

        vm.store(address(op), 0, bytes32(abi.encode(sender)));
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            message
        );
    }

    // relayMessage: should revert if eth is sent from a contract other than the standard bridge
    function test_replayMessage_withValue_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.expectRevert(
            "CrossDomainMessenger: value must be zero unless message is from a system address"
        );
        L1Messenger.relayMessage{ value: 100 }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
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
        L1Messenger.xDomainMessageSender();

        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(op));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            address(0),
            address(0),
            0,
            0,
            hex""
        );

        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L1Messenger.xDomainMessageSender();
    }

    // relayMessage: should send a successful call to the target contract after the first message
    // fails and ETH gets stuck, but the second message succeeds
    function test_relayMessage_retryAfterFailure_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        uint256 value = 100;

        vm.expectCall(target, hex"1111");

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.etch(target, address(new Reverter()).code);
        vm.deal(address(op), value);
        vm.prank(address(op));
        L1Messenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(L1Messenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(L1Messenger.successfulMessages(hash), false);
        assertEq(L1Messenger.failedMessages(hash), true);

        vm.expectEmit(true, true, true, true);

        emit RelayedMessage(hash);

        vm.etch(target, address(0).code);
        vm.prank(address(sender));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(L1Messenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(L1Messenger.successfulMessages(hash), true);
        assertEq(L1Messenger.failedMessages(hash), true);
    }

    function test_relayMessage_legacy_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // Compute the message hash.
        bytes32 hash = Hashing.hashCrossDomainMessageV1(
            // Using a legacy nonce with version 0.
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }),
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(hash);

        // Relay the message.
        vm.prank(address(op));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(L1Messenger.successfulMessages(hash), true);
        assertEq(L1Messenger.failedMessages(hash), false);
    }

    function test_relayMessage_legacyOldReplay_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // Compute the message hash.
        bytes32 hash = Hashing.hashCrossDomainMessageV1(
            // Using a legacy nonce with version 0.
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }),
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Mark legacy message as already relayed.
        uint256 successfulMessagesSlot = 203;
        bytes32 oldHash = Hashing.hashCrossDomainMessageV0(target, sender, hex"1111", 0);
        bytes32 slot = keccak256(abi.encode(oldHash, successfulMessagesSlot));
        vm.store(address(L1Messenger), slot, bytes32(uint256(1)));

        // Expect revert.
        vm.expectRevert("CrossDomainMessenger: legacy withdrawal already relayed");

        // Relay the message.
        vm.prank(address(op));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // Message was not relayed.
        assertEq(L1Messenger.successfulMessages(hash), false);
        assertEq(L1Messenger.failedMessages(hash), false);
    }

    function test_relayMessage_legacyRetryAfterFailure_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        uint256 value = 100;

        // Compute the message hash.
        bytes32 hash = Hashing.hashCrossDomainMessageV1(
            // Using a legacy nonce with version 0.
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }),
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Turn the target into a Reverter.
        vm.etch(target, address(new Reverter()).code);

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect FailedRelayedMessage event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit FailedRelayedMessage(hash);

        // Relay the message.
        vm.deal(address(op), value);
        vm.prank(address(op));
        L1Messenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message failed.
        assertEq(address(L1Messenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(L1Messenger.successfulMessages(hash), false);
        assertEq(L1Messenger.failedMessages(hash), true);

        // Make the target not revert anymore.
        vm.etch(target, address(0).code);

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(hash);

        // Retry the message.
        vm.prank(address(sender));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(L1Messenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(L1Messenger.successfulMessages(hash), true);
        assertEq(L1Messenger.failedMessages(hash), true);
    }

    function test_relayMessage_legacyRetryAfterSuccess_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        uint256 value = 100;

        // Compute the message hash.
        bytes32 hash = Hashing.hashCrossDomainMessageV1(
            // Using a legacy nonce with version 0.
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }),
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(hash);

        // Relay the message.
        vm.deal(address(op), value);
        vm.prank(address(op));
        L1Messenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(L1Messenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(L1Messenger.successfulMessages(hash), true);
        assertEq(L1Messenger.failedMessages(hash), false);

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");

        // Retry the message.
        vm.prank(address(sender));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );
    }

    function test_relayMessage_legacyRetryAfterFailureThenSuccess_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        uint256 value = 100;

        // Compute the message hash.
        bytes32 hash = Hashing.hashCrossDomainMessageV1(
            // Using a legacy nonce with version 0.
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }),
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Turn the target into a Reverter.
        vm.etch(target, address(new Reverter()).code);

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Relay the message.
        vm.deal(address(op), value);
        vm.prank(address(op));
        L1Messenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message failed.
        assertEq(address(L1Messenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(L1Messenger.successfulMessages(hash), false);
        assertEq(L1Messenger.failedMessages(hash), true);

        // Make the target not revert anymore.
        vm.etch(target, address(0).code);

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(true, true, true, true);
        emit RelayedMessage(hash);

        // Retry the message
        vm.prank(address(sender));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(L1Messenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(L1Messenger.successfulMessages(hash), true);
        assertEq(L1Messenger.failedMessages(hash), true);

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: message has already been relayed");

        // Retry the message again.
        vm.prank(address(sender));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );
    }
}
