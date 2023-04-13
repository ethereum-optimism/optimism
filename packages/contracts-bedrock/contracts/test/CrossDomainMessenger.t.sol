// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Messenger_Initializer, Reverter, CallerCaller } from "./CommonTest.t.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";

// Libraries
import { Predeploys } from "../libraries/Predeploys.sol";
import { Hashing } from "../libraries/Hashing.sol";
import { Encoding } from "../libraries/Encoding.sol";

// CrossDomainMessenger_Test is for testing functionality which is common to both the L1 and L2
// CrossDomainMessenger contracts. For simplicity, we use the L1 Messenger as the test contract.
contract CrossDomainMessenger_BaseGas_Test is Messenger_Initializer {
    // Ensure that baseGas passes for the max value of _minGasLimit,
    // this is about 4 Billion.
    function test_baseGas_succeeds() external view {
        L1Messenger.baseGas(hex"ff", type(uint32).max);
    }

    // Fuzz for other values which might cause a revert in baseGas.
    function testFuzz_baseGas_succeeds(uint32 _minGasLimit) external {
        L1Messenger.baseGas(hex"ff", _minGasLimit);
    }
}

contract CrossDomainMessenger_RelayMessage_Test is Messenger_Initializer {
    // Storage slot of the l2Sender
    uint256 constant senderSlotIndex = 50;

    // Nested call to relayMessage
    address[3] public xDomainMsgSenders;

    function nestedRelayMessage() external {
        xDomainMsgSenders[1] = L1Messenger.xDomainMessageSender();
    }

    function relayMessage() external payable {
        xDomainMsgSenders[0] = L1Messenger.xDomainMessageSender();

        address target = address(this);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory callMessage = abi.encodeWithSelector(CrossDomainMessenger_RelayMessage_Test.nestedRelayMessage.selector);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            callMessage
        );
        vm.prank(address(op));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            callMessage
        );

        xDomainMsgSenders[2] = L1Messenger.xDomainMessageSender();
    }


    // Ensure that the xdm messenger returns the expected xDomainMsgSender
    // given the level of relayMessage nested calls.
    function test_ReenteredRelayMessage() external {
        address target = address(this);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory callMessage = abi.encodeWithSelector(CrossDomainMessenger_RelayMessage_Test.relayMessage.selector);

        vm.expectCall(target, callMessage);

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
            callMessage
        );

        emit RelayedMessage(hash);

        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            callMessage
        );

        assert(L1Messenger.successfulMessages(hash));
        assertEq(L1Messenger.failedMessages(hash), false);

        assertEq(xDomainMsgSenders[0], sender);
        assertEq(xDomainMsgSenders[1], sender);
        assertEq(xDomainMsgSenders[2], sender);
    }
}
