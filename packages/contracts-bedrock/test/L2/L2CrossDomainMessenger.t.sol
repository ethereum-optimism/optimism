// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { stdError } from "forge-std/StdError.sol";

// Libraries
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";
import { Types } from "src/libraries/Types.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";

// Interfaces
import { IL2CrossDomainMessenger } from "interfaces/L2/IL2CrossDomainMessenger.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";

/// @title L2CrossDomainMessenger_TestInit
/// @notice Reusable test initialization for `L2CrossDomainMessenger` tests.
abstract contract L2CrossDomainMessenger_TestInit is CommonTest {
    /// @notice Receiver address for testing
    address recipient = address(0xabbaacdc);
}

/// @title L2CrossDomainMessenger_Constructor_Test
/// @notice Tests the `constructor` of the `L2CrossDomainMessenger` contract.
contract L2CrossDomainMessenger_Constructor_Test is L2CrossDomainMessenger_TestInit {
    /// @notice Tests that the implementation is initialized correctly.
    function test_constructor_succeeds() external view {
        IL2CrossDomainMessenger impl = IL2CrossDomainMessenger(
            EIP1967Helper.getImplementation(artifacts.mustGetAddress("L2CrossDomainMessenger"))
        );
        assertEq(address(impl.OTHER_MESSENGER()), address(0));
        assertEq(address(impl.otherMessenger()), address(0));
        assertEq(address(impl.l1CrossDomainMessenger()), address(0));
        assertGt(bytes(impl.version()).length, 0);
        assertEq(impl.MESSAGE_VERSION(), 1);
    }
}

/// @title L2CrossDomainMessenger_Initialize_Test
/// @notice Tests the `initialize` function of the `L2CrossDomainMessenger` contract.
contract L2CrossDomainMessenger_Initialize_Test is L2CrossDomainMessenger_TestInit {
    /// @notice Tests that the proxy is initialized correctly.
    function test_initialize_succeeds() external view {
        assertEq(address(l2CrossDomainMessenger.OTHER_MESSENGER()), address(l1CrossDomainMessenger));
        assertEq(address(l2CrossDomainMessenger.otherMessenger()), address(l1CrossDomainMessenger));
        assertEq(address(l2CrossDomainMessenger.l1CrossDomainMessenger()), address(l1CrossDomainMessenger));
        assertGt(bytes(l2CrossDomainMessenger.version()).length, 0);
        assertEq(l2CrossDomainMessenger.MESSAGE_VERSION(), 1);
        assertGt(l2CrossDomainMessenger.messageNonce(), 0);
    }
}

/// @title L2CrossDomainMessenger_SendMessage_Test
/// @notice Tests the `sendMessage` function of the `L2CrossDomainMessenger` contract.
contract L2CrossDomainMessenger_SendMessage_Test is L2CrossDomainMessenger_TestInit {
    /// @notice Tests that `sendMessage` executes successfully with various target addresses and gas limits.
    function testFuzz_sendMessage_withValidTargetAndGasLimit_succeeds(address _target, uint32 _minGasLimit) external {
        vm.assume(_target != address(0));
        _minGasLimit = uint32(bound(_minGasLimit, 21000, 30_000_000));

        uint256 initialNonce = l2CrossDomainMessenger.messageNonce();

        vm.prank(alice);
        l2CrossDomainMessenger.sendMessage(_target, hex"1234", _minGasLimit);

        assertEq(l2CrossDomainMessenger.messageNonce(), initialNonce + 1);
    }

    /// @notice Tests that `sendMessage` executes successfully with the original test case.
    function test_sendMessage_succeeds() external {
        bytes memory xDomainCallData =
            Encoding.encodeCrossDomainMessage(l2CrossDomainMessenger.messageNonce(), alice, recipient, 0, 100, hex"ff");
        vm.expectCall(
            address(l2ToL1MessagePasser),
            abi.encodeCall(
                IL2ToL1MessagePasser.initiateWithdrawal,
                (address(l1CrossDomainMessenger), l2CrossDomainMessenger.baseGas(hex"ff", 100), xDomainCallData)
            )
        );

        vm.expectEmit(true, true, true, true);
        emit MessagePassed(
            l2ToL1MessagePasser.messageNonce(),
            address(l2CrossDomainMessenger),
            address(l1CrossDomainMessenger),
            0,
            l2CrossDomainMessenger.baseGas(hex"ff", 100),
            xDomainCallData,
            Hashing.hashWithdrawal(
                Types.WithdrawalTransaction({
                    nonce: l2ToL1MessagePasser.messageNonce(),
                    sender: address(l2CrossDomainMessenger),
                    target: address(l1CrossDomainMessenger),
                    value: 0,
                    gasLimit: l2CrossDomainMessenger.baseGas(hex"ff", 100),
                    data: xDomainCallData
                })
            )
        );

        vm.prank(alice);
        l2CrossDomainMessenger.sendMessage(recipient, hex"ff", uint32(100));
    }

    /// @notice Tests that `sendMessage` can be called twice and that the nonce increments correctly.
    function test_sendMessage_twice_succeeds() external {
        uint256 nonce = l2CrossDomainMessenger.messageNonce();
        l2CrossDomainMessenger.sendMessage(recipient, hex"aa", uint32(500_000));
        l2CrossDomainMessenger.sendMessage(recipient, hex"aa", uint32(500_000));
        assertEq(nonce + 2, l2CrossDomainMessenger.messageNonce());
    }
}

/// @title L2CrossDomainMessenger_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the
///         `L2CrossDomainMessenger` contract.
contract L2CrossDomainMessenger_Uncategorized_Test is L2CrossDomainMessenger_TestInit {
    /// @notice Tests that `messageNonce` can be decoded correctly.
    function test_messageVersion_succeeds() external view {
        (, uint16 version) = Encoding.decodeVersionedNonce(l2CrossDomainMessenger.messageNonce());
        assertEq(version, l2CrossDomainMessenger.MESSAGE_VERSION());
    }

    /// @notice Tests that `sendMessage` reverts if the recipient is the zero address.
    function test_xDomainSender_senderNotSet_reverts() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l2CrossDomainMessenger.xDomainMessageSender();
    }

    /// @notice Tests that `sendMessage` reverts if the message version is not supported.
    function test_relayMessage_v2_reverts() external {
        address target = address(0xabcd);
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: only version 0 or 1 messages are supported at this time");

        // Try to relay a v2 message.
        vm.prank(caller);
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 2), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );
    }

    /// @notice Tests that `relayMessage` executes successfully.
    function test_relayMessage_succeeds() external {
        address target = address(0xabcd);
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        vm.expectCall(target, hex"1111");

        vm.prank(caller);

        vm.expectEmit(true, true, true, true);

        bytes32 hash =
            Hashing.hashCrossDomainMessage(Encoding.encodeVersionedNonce(0, 1), sender, target, 0, 0, hex"1111");

        emit RelayedMessage(hash);

        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // the message hash is in the successfulMessages mapping
        assert(l2CrossDomainMessenger.successfulMessages(hash));
        // it is not in the received messages mapping
        assertEq(l2CrossDomainMessenger.failedMessages(hash), false);
    }

    /// @notice Tests that `relayMessage` reverts if the value sent does not match the amount
    function test_relayMessage_fromOtherMessengerValueMismatch_reverts() external {
        // set the target to be alice
        address target = alice;
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        bytes memory message = hex"1111";

        // cannot send a message where the amount inputted does not match the msg.value
        vm.deal(caller, 10 ether);
        vm.prank(caller);
        vm.expectRevert(stdError.assertionError);
        l2CrossDomainMessenger.relayMessage{ value: 10 ether }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 9 ether, 0, message
        );
    }

    /// @notice Tests that `relayMessage` reverts if a failed message is attempted to be replayed
    ///         and the caller is the other messenger
    function test_relayMessage_fromOtherMessengerFailedMessageReplay_reverts() external {
        // set the target to be alice
        address target = alice;
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        bytes memory message = hex"1111";

        // make a failed message
        vm.etch(target, hex"fe");
        vm.prank(caller);
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );

        // cannot replay messages when the caller is the other messenger
        vm.prank(caller);
        vm.expectRevert(stdError.assertionError);
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );
    }

    /// @notice Tests that `relayMessage` reverts if attempting to relay a message sent to self
    function test_relayMessage_toSelf_reverts() external {
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(0), bytes32(abi.encode(sender)));

        vm.prank(caller);
        vm.expectRevert("CrossDomainMessenger: cannot send message to blocked system address");
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            address(l2CrossDomainMessenger),
            0,
            0,
            message
        );
    }

    /// @notice Tests that `relayMessage` reverts if attempting to relay a message sent to the
    ///         `l2ToL1MessagePasser` address
    function test_relayMessage_toL2ToL1MessagePasser_reverts() external {
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(0), bytes32(abi.encode(sender)));

        vm.prank(caller);
        vm.expectRevert("CrossDomainMessenger: cannot send message to blocked system address");
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            address(l2ToL1MessagePasser),
            0,
            0,
            message
        );
    }

    /// @notice Tests that `relayMessage` reverts if the message called by non-`optimismPortal2`
    ///         but not a failed message
    function test_relayMessage_relayingNewMessageByExternalUser_reverts() external {
        address target = address(alice);
        address sender = address(l1CrossDomainMessenger);
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(0), bytes32(abi.encode(sender)));

        vm.prank(bob);
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );
    }

    /// @notice Tests that `relayMessage` correctly resets the `xDomainMessageSender` to the
    ///         original value after a message is relayed.
    function test_xDomainMessageSender_reset_succeeds() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l2CrossDomainMessenger.xDomainMessageSender();

        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        vm.prank(caller);
        l2CrossDomainMessenger.relayMessage(Encoding.encodeVersionedNonce(0, 1), address(0), address(0), 0, 0, hex"");

        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l2CrossDomainMessenger.xDomainMessageSender();
    }

    /// @notice Tests that `relayMessage` is able to send a successful call to the target contract
    ///         after the first message fails and ETH gets stuck, but the second message succeeds.
    function test_relayMessage_retry_succeeds() external {
        address target = address(0xabcd);
        address sender = address(l1CrossDomainMessenger);
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        uint256 value = 100;

        bytes32 hash =
            Hashing.hashCrossDomainMessage(Encoding.encodeVersionedNonce(0, 1), sender, target, value, 0, hex"1111");

        vm.mockCallRevert(target, bytes(hex"1111"), bytes(hex""));
        vm.deal(address(caller), value);
        vm.prank(caller);
        l2CrossDomainMessenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(l2CrossDomainMessenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(l2CrossDomainMessenger.successfulMessages(hash), false);
        assertEq(l2CrossDomainMessenger.failedMessages(hash), true);

        vm.expectEmit(true, true, true, true);

        emit RelayedMessage(hash);

        vm.clearMockedCalls();
        vm.prank(address(sender));
        l2CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(l2CrossDomainMessenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(l2CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l2CrossDomainMessenger.failedMessages(hash), true);
    }
}
