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
    function testFuzz_baseGas_succeeds(uint32 _minGasLimit) external view {
        L1Messenger.baseGas(hex"ff", _minGasLimit);
    }
}

// CrossDomainMessenger_RelayMessage_Test tests re-entrency of relayMessage.
contract CrossDomainMessenger_RelayMessage_Test is Messenger_Initializer {
    // Storage slot of the l2Sender
    uint256 constant senderSlotIndex = 50;

    address public fuzzedSender;
    address public sender;

    address public target;

    // Internal helper function to relay a message and perform assertions.
    function internalRelay(address innerSender) internal {
        assertEq(sender, L1Messenger.xDomainMessageSender());

        bytes memory callMessage = abi.encodeWithSelector(
            CrossDomainMessenger_RelayMessage_Test.relayMessage.selector
        );

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            innerSender,
            target,
            0,
            0,
            callMessage
        );
        vm.prank(address(op));
        L1Messenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            innerSender,
            target,
            0,
            0,
            callMessage
        );

        assertTrue(L1Messenger.failedMessages(hash));
        assertFalse(L1Messenger.successfulMessages(hash));
        assertEq(sender, L1Messenger.xDomainMessageSender());
    }

    // relayMessage is called by the L1 Cross Domain Messenger.
    function relayMessage() external payable {
        for (uint256 i = 0; i < 10; i++) {
            address innerSender;
            unchecked {
                innerSender = address(uint160(uint256(uint160(fuzzedSender)) + i));
            }
            internalRelay(innerSender);
        }
    }

    function testFuzz_relayMessageReenter_succeeds(address _sender) external {
        vm.assume(_sender != Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        fuzzedSender = _sender;

        target = address(this);
        sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory callMessage = abi.encodeWithSelector(
            CrossDomainMessenger_RelayMessage_Test.relayMessage.selector
        );

        vm.expectCall(target, callMessage);

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            target,
            0,
            0,
            callMessage
        );

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(op), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(op));
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

        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        L1Messenger.xDomainMessageSender();
    }
}
