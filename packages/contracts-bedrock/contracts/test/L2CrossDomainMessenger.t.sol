//SPDX-License-Identifier: MIT
pragma solidity 0.8.10;

import { Messenger_Initializer } from "./CommonTest.t.sol";

import { AddressAliasHelper } from "../vendor/AddressAliasHelper.sol";
import { L2ToL1MessagePasser } from "../L2/L2ToL1MessagePasser.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { L2CrossDomainMessenger } from "../L2/L2CrossDomainMessenger.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

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
        assertEq(
            version,
            L2Messenger.MESSAGE_VERSION()
        );
    }

    function test_L2MessengerSendMessage() external {
        vm.expectCall(
            address(messagePasser),
            abi.encodeWithSelector(
                L2ToL1MessagePasser.initiateWithdrawal.selector,
                address(L1Messenger),
                100 + L2Messenger.baseGas(hex"ff"),
                Encoding.encodeCrossDomainMessage(
                    L2Messenger.messageNonce(),
                    alice,
                    recipient,
                    0,
                    100,
                    hex"ff"
                )
            )
        );

        // WithdrawalInitiated event
        vm.expectEmit(true, true, true, true);
        emit WithdrawalInitiated(
            messagePasser.nonce(),
            address(L2Messenger),
            address(L1Messenger),
            0,
            100 + L2Messenger.baseGas(hex"ff"),
            Encoding.encodeCrossDomainMessage(
                L2Messenger.messageNonce(),
                alice,
                recipient,
                0,
                100,
                hex"ff"
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
        assertEq(
            nonce + 2,
            L2Messenger.messageNonce()
        );
    }

    function test_L2MessengerXDomainSenderReverts() external {
        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
    }

    function test_L2MessengerRelayMessageSucceeds() external {
        address target = address(0xabcd);
        address sender = address(L1Messenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));

        vm.expectCall(target, hex"1111");

        vm.prank(caller);

        vm.expectEmit(true, true, true, true);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            0,
            sender,
            target,
            0,
            0,
            hex"1111"
        );

        emit RelayedMessage(hash);

        L2Messenger.relayMessage(
            0, // nonce
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
        vm.expectRevert("Message cannot be replayed.");
        L1Messenger.relayMessage(0, sender, target, 0, 0, message);
    }

    // relayMessage: the xDomainMessageSender is reset to the original value
    function test_L2MessengerxDomainMessageSenderResets() external {
        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();

        address caller = AddressAliasHelper.applyL1ToL2Alias(address(L1Messenger));
        vm.prank(caller);
        L2Messenger.relayMessage(0, address(0), address(0), 0, 0, hex"");

        vm.expectRevert("xDomainMessageSender is not set");
        L2Messenger.xDomainMessageSender();
    }

    // relayMessage: should revert if paused
    function test_L2MessengerRelayShouldRevertIfPaused() external {
        vm.prank(L2Messenger.owner());
        L2Messenger.pause();

        vm.expectRevert("Pausable: paused");
        L2Messenger.relayMessage(0, address(0), address(0), 0, 0, hex"");
    }
}
