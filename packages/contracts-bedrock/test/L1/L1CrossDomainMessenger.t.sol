// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing utilities
import { CommonTest } from "test/setup/CommonTest.sol";
import { GasBurner } from "test/mocks/GasBurner.sol";
import { stdError } from "forge-std/StdError.sol";
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

// Libraries
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";

// Target contract dependencies
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";

/// @title L1CrossDomainMessenger_Encoding_Harness
/// @notice A harness contract for testing internal functions of the Encoding library.
contract L1CrossDomainMessenger_Encoding_Harness {
    function encodeCrossDomainMessage(
        uint256 nonce,
        address sender,
        address target,
        uint256 value,
        uint256 gasLimit,
        bytes memory data
    )
        external
        pure
        returns (bytes memory)
    {
        return Encoding.encodeCrossDomainMessage(nonce, sender, target, value, gasLimit, data);
    }
}

/// @title L1CrossDomainMessenger_TestInit
/// @notice Reusable test initialization for L1CrossDomainMessenger tests.
abstract contract L1CrossDomainMessenger_TestInit is CommonTest {
    /// @dev The receiver address
    address recipient = address(0xabbaacdc);

    /// @dev The storage slot of the l2Sender
    uint256 senderSlotIndex;

    /// @dev Encoding library harness.
    L1CrossDomainMessenger_Encoding_Harness encoding;

    function setUp() public virtual override {
        super.setUp();
        senderSlotIndex = ForgeArtifacts.getSlot("OptimismPortal2", "l2Sender").slot;
        encoding = new L1CrossDomainMessenger_Encoding_Harness();
    }
}

/// @title L1CrossDomainMessenger_Constructor_Test
/// @notice Tests for the `constructor` of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_Constructor_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that the implementation is initialized correctly.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_constructor_succeeds() external virtual {
        IL1CrossDomainMessenger impl = IL1CrossDomainMessenger(addressManager.getAddress("OVM_L1CrossDomainMessenger"));
        assertEq(address(impl.systemConfig()), address(0));
        assertEq(address(impl.PORTAL()), address(0));
        assertEq(address(impl.portal()), address(0));

        // The constructor now uses _disableInitializers, whereas OP Mainnet has the other
        // messenger in storage
        returnIfForkTest("L1CrossDomainMessenger_Test: impl storage differs on forked network");
        assertEq(address(impl.OTHER_MESSENGER()), address(0));
        assertEq(address(impl.otherMessenger()), address(0));
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotResolvedDelegateProxy.selector);
        impl.proxyAdmin();
    }
}

/// @title L1CrossDomainMessenger_Initialize_Test
/// @notice Tests for the `initialize` function of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_Initialize_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that the proxy is initialized correctly.
    function test_initialize_succeeds() external view {
        assertEq(address(l1CrossDomainMessenger.systemConfig()), address(systemConfig));
        assertEq(address(l1CrossDomainMessenger.PORTAL()), address(optimismPortal2));
        assertEq(address(l1CrossDomainMessenger.portal()), address(optimismPortal2));
        assertEq(address(l1CrossDomainMessenger.OTHER_MESSENGER()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assertEq(address(l1CrossDomainMessenger.otherMessenger()), Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        assertEq(address(l1CrossDomainMessenger.proxyAdmin()), address(proxyAdmin));
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal
    ///         initialization but confirms that the initValue is not incremented incorrectly if
    ///         an upgrade function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("L1CrossDomainMessenger", "_initialized");

        // Get the initializer value.
        // Note that for L1CrossDomainMessenger the initialized value is stored at offset 20 so
        // this test is slightly different from other similar tests.
        bytes32 slotVal = vm.load(address(l1CrossDomainMessenger), bytes32(slot.slot));
        uint8 val = uint8((uint256(slotVal) >> 20 * 8) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, l1CrossDomainMessenger.initVersion());
    }

    /// @notice Tests that the initialize function reverts if called by a non-proxy admin or owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("L1CrossDomainMessenger", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(l1CrossDomainMessenger), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender
        vm.prank(_sender);
        l1CrossDomainMessenger.initialize(systemConfig, optimismPortal2);
    }

    /// @notice Fuzz test for initialize with any system config address.
    /// @param _systemConfig The system config address to test.
    function testFuzz_initialize_anySystemConfig_succeeds(address _systemConfig) external {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("L1CrossDomainMessenger", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(l1CrossDomainMessenger), bytes32(slot.slot), bytes32(0));

        // Initialize with the fuzzed system config address
        vm.prank(address(proxyAdmin));
        l1CrossDomainMessenger.initialize(ISystemConfig(_systemConfig), optimismPortal2);

        // Verify the address was set correctly
        assertEq(address(l1CrossDomainMessenger.systemConfig()), _systemConfig);
        assertEq(address(l1CrossDomainMessenger.portal()), address(optimismPortal2));
    }

    /// @notice Fuzz test for initialize with any portal address.
    /// @param _portal The portal address to test.
    function testFuzz_initialize_anyPortal_succeeds(address _portal) external {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("L1CrossDomainMessenger", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(l1CrossDomainMessenger), bytes32(slot.slot), bytes32(0));

        // Initialize with the fuzzed portal address
        vm.prank(address(proxyAdmin));
        l1CrossDomainMessenger.initialize(systemConfig, IOptimismPortal2(payable(_portal)));

        // Verify the address was set correctly
        assertEq(address(l1CrossDomainMessenger.systemConfig()), address(systemConfig));
        assertEq(address(l1CrossDomainMessenger.portal()), _portal);
    }
}

/// @title L1CrossDomainMessenger_Paused_Test
/// @notice Tests for the `paused` functionality of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_Paused_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that the superchain config is called by the messenger's paused function.
    function test_pause_callsSuperchainConfig_succeeds() external {
        // We use abi.encodeWithSignature because paused is overloaded.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.expectCall(address(superchainConfig), abi.encodeWithSignature("paused(address)", address(0)));
        l1CrossDomainMessenger.paused();
    }

    /// @notice Tests that changing the superchain config paused status changes the return value
    ///         of the messenger.
    function test_pause_matchesSuperchainConfig_succeeds() external {
        assertFalse(l1CrossDomainMessenger.paused());
        assertEq(l1CrossDomainMessenger.paused(), superchainConfig.paused(address(0)));

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        assertTrue(l1CrossDomainMessenger.paused());
        assertEq(l1CrossDomainMessenger.paused(), superchainConfig.paused(address(0)));
    }
}

/// @title L1CrossDomainMessenger_SuperchainConfig_Test
/// @notice Tests for the `superchainConfig` function of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_SuperchainConfig_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that `superchainConfig` returns the correct address.
    function test_superchainConfig_succeeds() external view {
        assertEq(address(l1CrossDomainMessenger.superchainConfig()), address(superchainConfig));
    }
}

/// @title L1CrossDomainMessenger_portal_Test
/// @notice Tests for the `PORTAL` legacy getter function of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_portal_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that `PORTAL` returns the correct portal address.
    function test_portal_succeeds() external view {
        assertEq(address(l1CrossDomainMessenger.PORTAL()), address(optimismPortal2));
        assertEq(address(l1CrossDomainMessenger.PORTAL()), address(l1CrossDomainMessenger.portal()));
    }
}

/// @notice The following tests are not testing any function of the L1CrossDomainMessenger
///         contract directly, but are testing the functionality of the CrossDomainMessenger
///         contract that is inherited from.

/// @title L1CrossDomainMessenger_SendMessage_Test
/// @notice Tests for the `sendMessage` functionality of the L1CrossDomainMessenger.
contract L1CrossDomainMessenger_SendMessage_Test is L1CrossDomainMessenger_TestInit {
    /// @notice Tests that the `sendMessage` function is able to send a single message.
    /// TODO: this same test needs to be done with the legacy message type
    ///       by setting the message version to 0
    function test_sendMessage_succeeds() external {
        // deposit transaction on the optimism portal should be called
        vm.expectCall(
            address(optimismPortal2),
            abi.encodeCall(
                IOptimismPortal2.depositTransaction,
                (
                    Predeploys.L2_CROSS_DOMAIN_MESSENGER,
                    0,
                    l1CrossDomainMessenger.baseGas(hex"ff", 100),
                    false,
                    Encoding.encodeCrossDomainMessage(
                        l1CrossDomainMessenger.messageNonce(), alice, recipient, 0, 100, hex"ff"
                    )
                )
            )
        );

        // TransactionDeposited event
        vm.expectEmit(address(optimismPortal2));
        emitTransactionDeposited(
            AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger)),
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            0,
            0,
            l1CrossDomainMessenger.baseGas(hex"ff", 100),
            false,
            Encoding.encodeCrossDomainMessage(l1CrossDomainMessenger.messageNonce(), alice, recipient, 0, 100, hex"ff")
        );

        // SentMessage event
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(recipient, alice, hex"ff", l1CrossDomainMessenger.messageNonce(), 100);

        // SentMessageExtension1 event
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessageExtension1(alice, 0);

        vm.prank(alice);
        l1CrossDomainMessenger.sendMessage(recipient, hex"ff", uint32(100));
    }

    /// @notice Fuzz test for sendMessage with various gas limits and message data.
    /// @param _gasLimit Gas limit for the message (bounded to reasonable range).
    /// @param _message Message data to send.
    /// @param _sender Address sending the message.
    function testFuzz_sendMessage_varyingInputs_succeeds(
        uint32 _gasLimit,
        bytes calldata _message,
        address _sender
    )
        external
    {
        // Bound gas limit to reasonable range to avoid OutOfGas errors
        _gasLimit = uint32(bound(uint256(_gasLimit), 21000, 1_000_000));
        // Bound message length to avoid excessive gas costs
        vm.assume(_message.length <= 1000);
        vm.assume(_sender != address(0));

        uint256 nonceBefore = l1CrossDomainMessenger.messageNonce();

        vm.prank(_sender);
        l1CrossDomainMessenger.sendMessage(recipient, _message, _gasLimit);

        // Verify nonce incremented
        assertEq(l1CrossDomainMessenger.messageNonce(), nonceBefore + 1);
    }

    /// @notice Tests that the sendMessage function is able to send the same message twice.
    function test_sendMessage_twice_succeeds() external {
        uint256 nonce = l1CrossDomainMessenger.messageNonce();
        l1CrossDomainMessenger.sendMessage(recipient, hex"aa", uint32(500_000));
        l1CrossDomainMessenger.sendMessage(recipient, hex"aa", uint32(500_000));
        // the nonce increments for each message sent
        assertEq(nonce + 2, l1CrossDomainMessenger.messageNonce());
    }

    /// @notice Tests sendMessage with zero gas limit.
    function test_sendMessage_zeroGasLimit_succeeds() external {
        uint256 nonce = l1CrossDomainMessenger.messageNonce();

        // Even with zero gas limit, message should send
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(recipient, alice, hex"1234", nonce, 0);

        vm.prank(alice);
        l1CrossDomainMessenger.sendMessage(recipient, hex"1234", 0);

        // Verify nonce incremented
        assertEq(l1CrossDomainMessenger.messageNonce(), nonce + 1);
    }

    /// @notice Tests sendMessage with high gas limit that causes OutOfGas.
    function test_sendMessage_highGasLimit_reverts() external {
        // Very high gas limit causes OutOfGas error in portal deposit
        uint32 highGasLimit = 30_000_000;

        vm.prank(alice);
        vm.expectRevert("OutOfGas()");
        l1CrossDomainMessenger.sendMessage(recipient, hex"5678", highGasLimit);
    }
}

/// @title L1CrossDomainMessenger_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the L1CrossDomainMessenger
///         but are testing functionality of the CrossDomainMessenger contract that is inherited
///         from.
contract L1CrossDomainMessenger_Uncategorized_Test is L1CrossDomainMessenger_TestInit {
    // Variables for reentrancy test
    bool reentrancyAttacked;
    uint256 constant reentrancyMessageValue = 50;
    bytes reentrancySelector;
    address reentrancySender;
    bytes32 reentrancyHash;
    address reentrancyTarget;

    function setUp() public virtual override {
        super.setUp();
        // Setup for reentrancy test variables (balance setup moved to specific test)
        reentrancyTarget = address(this);
        reentrancySender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        reentrancySelector = abi.encodeCall(this.reinitAndReenter, ());
        reentrancyHash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            reentrancySender,
            reentrancyTarget,
            reentrancyMessageValue,
            0,
            reentrancySelector
        );
    }

    /// @notice This method will be called by the relayed message, and will attempt to reenter the
    ///         `relayMessage` function exactly one.
    function reinitAndReenter() external payable {
        // only attempt the attack once
        if (!reentrancyAttacked) {
            reentrancyAttacked = true;
            // set initialized to false
            vm.store(address(l1CrossDomainMessenger), 0, bytes32(uint256(0)));

            // call the initializer function
            l1CrossDomainMessenger.initialize(ISystemConfig(systemConfig), IOptimismPortal2(optimismPortal2));

            // attempt to re-replay the withdrawal
            vm.expectEmit(address(l1CrossDomainMessenger));
            emit FailedRelayedMessage(reentrancyHash);
            l1CrossDomainMessenger.relayMessage(
                Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
                reentrancySender,
                reentrancyTarget,
                reentrancyMessageValue,
                0,
                reentrancySelector
            );
        }
    }

    /// @notice Tests that the version can be decoded from the message nonce.
    function test_messageVersion_succeeds() external view {
        (, uint16 version) = Encoding.decodeVersionedNonce(l1CrossDomainMessenger.messageNonce());
        assertEq(version, l1CrossDomainMessenger.MESSAGE_VERSION());
    }

    /// @notice Tests that the xDomainMessageSender reverts when not set.
    function test_xDomainSender_notSet_reverts() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l1CrossDomainMessenger.xDomainMessageSender();
    }

    /// @notice Tests that the `relayMessage` function reverts when the message version is not 0 or
    ///         1.
    /// @notice Marked virtual to be overridden in
    ///         test/kontrol/deployment/DeploymentSummary.t.sol
    function test_relayMessage_v2_reverts() external virtual {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // Set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: only version 0 or 1 messages are supported at this time");

        // Try to relay a v2 message.
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 2 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );
    }

    /// @notice Tests that the `relayMessage` function is able to relay a message
    ///         successfully by calling the target contract.
    function test_relayMessage_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        vm.expectCall(target, hex"1111");

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(optimismPortal2));

        vm.expectEmit(address(l1CrossDomainMessenger));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, hex"1111"
        );

        emit RelayedMessage(hash);

        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // the message hash is in the successfulMessages mapping
        assert(l1CrossDomainMessenger.successfulMessages(hash));
        // it is not in the received messages mapping
        assertEq(l1CrossDomainMessenger.failedMessages(hash), false);
    }

    /// @notice Fuzz test for relaying messages with various parameters.
    /// @param _target Target address for the message.
    /// @param _minGasLimit Minimum gas limit for message execution.
    /// @param _message Message data to relay.
    function testFuzz_relayMessage_varyingInputs_succeeds(
        address _target,
        uint32 _minGasLimit,
        bytes calldata _message
    )
        external
    {
        // Ensure target is not a blocked address
        vm.assume(_target != address(l1CrossDomainMessenger));
        vm.assume(_target != address(optimismPortal2));
        vm.assume(_target != address(0));

        // Bound gas limit and message size to avoid OutOfGas errors
        _minGasLimit = uint32(bound(uint256(_minGasLimit), 0, 100_000));
        vm.assume(_message.length <= 100);

        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, _target, 0, _minGasLimit, _message
        );

        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, _target, 0, _minGasLimit, _message
        );

        // Verify message was relayed (either successfully or failed)
        assertTrue(l1CrossDomainMessenger.successfulMessages(hash) || l1CrossDomainMessenger.failedMessages(hash));
    }

    /// @notice Tests that `relayMessage` reverts if the caller is optimismPortal2 and the value
    ///         sent does not match the amount.
    function test_relayMessage_fromOtherMessengerValueMismatch_reverts() external {
        address target = alice;
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        // set the value of op.l2Sender() to be the L2CrossDomainMessenger.
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // correctly sending as OptimismPortal but amount does not match msg.value
        vm.deal(address(optimismPortal2), 10 ether);
        vm.prank(address(optimismPortal2));
        vm.expectRevert(stdError.assertionError);
        l1CrossDomainMessenger.relayMessage{ value: 10 ether }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 9 ether, 0, message
        );
    }

    /// @notice Tests that `relayMessage` reverts if a failed message is attempted to be replayed
    ///         via the optimismPortal2.
    function test_relayMessage_fromOtherMessengerFailedMessageReplay_reverts() external {
        address target = alice;
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        // set the value of op.l2Sender() to be the L2 Cross Domain Messenger.
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // make a failed message
        vm.etch(target, hex"fe");
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );

        // cannot replay messages when optimism portal is msg.sender
        vm.prank(address(optimismPortal2));
        vm.expectRevert(stdError.assertionError);
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );
    }

    /// @notice Tests that `relayMessage` reverts if attempting to relay a message with
    ///         l1CrossDomainMessenger as the target.
    function test_relayMessage_toSelf_reverts() external {
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        vm.prank(address(optimismPortal2));
        vm.expectRevert("CrossDomainMessenger: cannot send message to blocked system address");
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }),
            sender,
            address(l1CrossDomainMessenger),
            0,
            0,
            message
        );
    }

    /// @notice Tests that `relayMessage` reverts if attempting to relay a message with
    ///         optimismPortal as the target.
    function test_relayMessage_toOptimismPortal_reverts() external {
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        vm.prank(address(optimismPortal2));
        vm.expectRevert("CrossDomainMessenger: cannot send message to blocked system address");
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, address(optimismPortal2), 0, 0, message
        );
    }

    /// @notice Tests that `relayMessage` reverts if the message is called by a non-optimismPortal2
    ///         address and is not a failed message eligible for replay.
    function test_relayMessage_relayingNewMessageByExternalUser_reverts() external {
        address target = address(alice);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        vm.prank(bob);
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );
    }

    /// @notice Tests that the `relayMessage` function on L2 will always succeed for any potential
    ///         message, regardless of the size of the message, the minimum gas limit, or the
    ///         amount of gas used by the target contract.
    function testFuzz_relayMessage_baseGasSufficient_succeeds(
        uint24 _messageLength,
        uint32 _minGasLimit,
        uint32 _gasToUse
    )
        external
    {
        // TODO(#14609): Update this test to use default.isolate = true once a new stable Foundry
        // release is available that includes #9904. That will allow us to use this test to check
        // for changes to the EVM itself that might cause our gas formula to be incorrect.

        // Skip if this is a fork test, won't work.
        skipIfForkTest("L2CrossDomainMessenger doesn't exist on L1 in forked test");

        // Using smaller uint so the fuzzer doesn't give as many massive values that get bounded.
        // TODO: Known issue, messages above 34k can actually OOG on the receiving side if the
        // target uses all available gas. Can be resolved by capping data sizes on the XDM or by
        // increasing the amount of available relay gas to ~100k. If increasing relay gas, should
        // have the relay gas only increase when the calldata size is large to avoid disrupting
        // in-flight L2 -> L1 messages.
        _messageLength = uint24(bound(_messageLength, 0, 34_000));

        // Need more than 500 since GasBurner requires it.
        // It's ok to try to use more than minGasLimit since the L2CrossDomainMessenger should
        // catch the revert and store the message hash in the failedMessages mapping.
        _gasToUse = uint32(bound(_gasToUse, 500, type(uint32).max));

        // Generate random bytes, more useful than having a _message parameter.
        bytes memory _message = vm.randomBytes(_messageLength);

        // Compute the base gas.
        // Base gas should really be computed on the fully encoded message but that would break the
        // expected API, so we instead just add the encoding overhead to the message length inside
        // of the baseGas function.
        uint64 baseGas = l1CrossDomainMessenger.baseGas(_message, _minGasLimit);

        // Deploy a gas burner.
        address target = address(new GasBurner(_gasToUse));

        // Encode the message.
        bytes memory encoded = Encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce(0, 1), // nonce
            alice, // Sender doesn't matter
            target,
            0, // Value doesn't matter
            _minGasLimit,
            _message
        );

        // Count the number of non-zero bytes in the message.
        uint256 zeroBytesInCalldata = 0;
        uint256 nonzeroBytesInCalldata = 0;
        for (uint256 i = 0; i < encoded.length; i++) {
            if (encoded[i] != bytes1(0)) {
                nonzeroBytesInCalldata++;
            } else {
                zeroBytesInCalldata++;
            }
        }

        // Base gas must always be sufficient to cover the floor cost from EIP-7623.
        assertGt(baseGas, 21000 + ((zeroBytesInCalldata + nonzeroBytesInCalldata * 4) * 10));

        // Actual gas on L2 will be the base gas minus the intrinsic gas cost. Note that even after
        // EIP-7623, we still deduct 21k + 16 gas per calldata token from the gas limit before
        // execution happens. After execution, if the message didn't spend enough in execution gas,
        // the EVM will floor the cost of the transaction to 21k + 40 gas per calldata token.
        uint256 gasSupplied = baseGas - (21000 + ((zeroBytesInCalldata + nonzeroBytesInCalldata * 4) * 4));

        // We'll trigger the L2CrossDomainMessenger as if we're the L1CrossDomainMessenger
        address caller = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));
        vm.prank(caller);

        // Trigger the L2CrossDomainMessenger.
        // Should NOT fail.
        (bool success,) = address(l2CrossDomainMessenger).call{ gas: gasSupplied }(encoded);
        assertTrue(success, "L2CrossDomainMessenger call should not fail");

        // Message should either be in the failed or successful messages mapping.
        bool inFailedMessages = l2CrossDomainMessenger.failedMessages(keccak256(encoded));
        bool inSuccessfulMessages = l2CrossDomainMessenger.successfulMessages(keccak256(encoded));
        assertTrue(
            inFailedMessages || inSuccessfulMessages, "message should be in either failed or successful messages"
        );
    }

    /// @notice Tests that the `relayMessage` function reverts if ETH is sent from a contract other
    ///         than the standard bridge.
    function test_replayMessage_withValue_reverts() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        bytes memory message = hex"1111";

        vm.expectRevert("CrossDomainMessenger: value must be zero unless message is from a system address");
        l1CrossDomainMessenger.relayMessage{ value: 100 }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, 0, 0, message
        );
    }

    /// @notice Tests that the `ENCODING_OVERHEAD` is always greater than or equal to the number of
    ///         bytes in an encoded message OTHER than the size of the data field.
    ///         `ENCODING_OVERHEAD` needs to account for these other bytes so that the total
    ///         message size used in the `baseGas` function is accurate. This test ensures that if
    ///         the encoding ever changes, this test will fail and the developer will need to
    ///         update the `ENCODING_OVERHEAD` constant.
    /// @param _nonce The nonce to encode into the message.
    /// @param _version The version to encode into the message.
    /// @param _sender The sender to encode into the message.
    /// @param _target The target to encode into the message.
    /// @param _value The value to encode into the message.
    /// @param _minGasLimit The minimum gas limit to encode into the message.
    /// @param _message The message to encode into the message.
    function testFuzz_encodingOverhead_sufficient_succeeds(
        uint240 _nonce,
        uint16 _version,
        address _sender,
        address _target,
        uint256 _value,
        uint256 _minGasLimit,
        bytes memory _message
    )
        external
    {
        // Make sure that unexpected nonces aren't being used right now.
        // Prevents people from forgetting to update this test if a new version is ever used.
        if (_version > 2) {
            vm.expectRevert("Encoding: unknown cross domain message version");
            encoding.encodeCrossDomainMessage(
                Encoding.encodeVersionedNonce({ _nonce: 0, _version: _version }),
                _sender,
                _target,
                _value,
                _minGasLimit,
                _message
            );
        }

        // Clamp the version to 0 or 1.
        _version = _version % 2;

        // Encode the message.
        bytes memory encoded = encoding.encodeCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: _nonce, _version: _version }),
            _sender,
            _target,
            _value,
            _minGasLimit,
            _message
        );

        // Total size should always be greater than or equal to the overhead bytes.
        assertGe(l1CrossDomainMessenger.ENCODING_OVERHEAD(), encoded.length - _message.length);
    }

    /// @notice Tests that the xDomainMessageSender is reset to the original value
    ///         after a message is relayed.
    function test_xDomainMessageSender_reset_succeeds() external {
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l1CrossDomainMessenger.xDomainMessageSender();

        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;

        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), address(0), address(0), 0, 0, hex""
        );

        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l1CrossDomainMessenger.xDomainMessageSender();
    }

    /// @notice Tests that xDomainMessageSender is never set during sendMessage.
    function test_xDomainMessageSender_duringSend_reverts() external {
        // XDomainMessageSender is only set during relayMessage, not sendMessage
        vm.expectRevert("CrossDomainMessenger: xDomainMessageSender is not set");
        l1CrossDomainMessenger.xDomainMessageSender();
    }

    /// @notice Tests that `relayMessage` should successfully call the target contract after the
    ///         first message fails and ETH is stuck, but the second message succeeds with a
    ///         version 1 message.
    function test_relayMessage_retryAfterFailure_succeeds() external {
        address target = address(0xabcd);
        address sender = Predeploys.L2_CROSS_DOMAIN_MESSENGER;
        uint256 value = 100;

        vm.expectCall(target, hex"1111");

        bytes32 hash = Hashing.hashCrossDomainMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), sender, target, value, 0, hex"1111"
        );

        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        vm.mockCallRevert(target, bytes(hex"1111"), bytes(hex""));
        vm.deal(address(optimismPortal2), value);
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(l1CrossDomainMessenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), false);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);

        vm.clearMockedCalls();

        vm.expectEmit(address(l1CrossDomainMessenger));
        emit RelayedMessage(hash);

        vm.etch(target, address(0).code);
        vm.prank(address(sender));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        assertEq(address(l1CrossDomainMessenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);
    }

    /// @notice Tests that `relayMessage` should successfully call the target contract after the
    ///         first message fails and ETH is stuck, but the second message succeeds with a
    ///         legacy message (version 0).
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
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit RelayedMessage(hash);

        // Relay the message.
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), false);
    }

    /// @notice Tests that `relayMessage` should revert if the message is already replayed.
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
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));
        // Mark legacy message as already relayed.
        uint256 successfulMessagesSlot = 203;
        bytes32 oldHash = Hashing.hashCrossDomainMessageV0(target, sender, hex"1111", 0);
        bytes32 slot = keccak256(abi.encode(oldHash, successfulMessagesSlot));
        vm.store(address(l1CrossDomainMessenger), slot, bytes32(uint256(1)));

        // Expect revert.
        vm.expectRevert("CrossDomainMessenger: legacy withdrawal already relayed");

        // Relay the message.
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            0, // value
            0,
            hex"1111"
        );

        // Message was not relayed.
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), false);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), false);
    }

    /// @notice Tests that `relayMessage` can be retried after a failure with a legacy message.
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
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Make the target revert.
        vm.mockCallRevert(target, bytes(hex"1111"), bytes(hex""));
        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect FailedRelayedMessage event to be emitted.
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit FailedRelayedMessage(hash);

        // Relay the message.
        vm.deal(address(optimismPortal2), value);
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message failed.
        assertEq(address(l1CrossDomainMessenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), false);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);

        // Make the target not revert anymore.
        vm.clearMockedCalls();

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit RelayedMessage(hash);

        // Retry the message.
        vm.prank(address(sender));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(l1CrossDomainMessenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);
    }

    /// @notice Tests that `relayMessage` cannot be retried after success with a legacy message.
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
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit RelayedMessage(hash);

        // Relay the message.
        vm.deal(address(optimismPortal2), value);
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(l1CrossDomainMessenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), false);

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");

        // Retry the message.
        vm.prank(address(sender));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );
    }

    /// @notice Tests that `relayMessage` cannot be called after a failure and a successful replay.
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
        vm.store(address(optimismPortal2), bytes32(senderSlotIndex), bytes32(abi.encode(sender)));

        // Make the target revert.
        vm.mockCallRevert(target, bytes(hex"1111"), bytes(hex""));

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Relay the message.
        vm.deal(address(optimismPortal2), value);
        vm.prank(address(optimismPortal2));
        l1CrossDomainMessenger.relayMessage{ value: value }(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message failed.
        assertEq(address(l1CrossDomainMessenger).balance, value);
        assertEq(address(target).balance, 0);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), false);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);

        // Make the target not revert anymore.
        vm.clearMockedCalls();

        // Target should be called with expected data.
        vm.expectCall(target, hex"1111");

        // Expect RelayedMessage event to be emitted.
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit RelayedMessage(hash);

        // Retry the message
        vm.prank(address(sender));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );

        // Message was successfully relayed.
        assertEq(address(l1CrossDomainMessenger).balance, 0);
        assertEq(address(target).balance, value);
        assertEq(l1CrossDomainMessenger.successfulMessages(hash), true);
        assertEq(l1CrossDomainMessenger.failedMessages(hash), true);

        // Expect a revert.
        vm.expectRevert("CrossDomainMessenger: message has already been relayed");

        // Retry the message again.
        vm.prank(address(sender));
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 0 }), // nonce
            sender,
            target,
            value,
            0,
            hex"1111"
        );
    }

    /// @notice Tests that the relayMessage function is able to relay a message successfully by
    ///         calling the target contract.
    function test_relayMessage_paused_reverts() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));
        vm.expectRevert("CrossDomainMessenger: paused");

        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            address(0),
            address(0),
            0, // value
            0,
            hex"1111"
        );
    }

    /// @notice A regression test against a reentrancy vulnerability in the `CrossDomainMessenger`
    ///         contract, which was possible by intercepting and sandwiching a signed Safe
    ///         Transaction to upgrade it. Tests that the `relayMessage` function cannot be
    ///         reentered by calling the `initialize` function within the relayed message.
    function test_relayMessage_replayStraddlingReinit_reverts() external {
        // Setup balance for this specific test
        vm.deal(address(l1CrossDomainMessenger), reentrancyMessageValue * 2);

        uint256 balanceBeforeThis = address(this).balance;
        uint256 balanceBeforeMessenger = address(l1CrossDomainMessenger).balance;

        // A requisite for the attack is that the message has already been attempted and written
        // to the failedMessages mapping, so that it can be replayed.
        vm.store(address(l1CrossDomainMessenger), keccak256(abi.encode(reentrancyHash, 206)), bytes32(uint256(1)));
        assertTrue(l1CrossDomainMessenger.failedMessages(reentrancyHash));

        vm.expectEmit(address(l1CrossDomainMessenger));
        emit FailedRelayedMessage(reentrancyHash);
        l1CrossDomainMessenger.relayMessage(
            Encoding.encodeVersionedNonce({ _nonce: 0, _version: 1 }), // nonce
            reentrancySender,
            reentrancyTarget,
            reentrancyMessageValue,
            0,
            reentrancySelector
        );

        // The message hash is not in the successfulMessages mapping.
        assertFalse(l1CrossDomainMessenger.successfulMessages(reentrancyHash));
        // The balance of this contract is unchanged.
        assertEq(address(this).balance, balanceBeforeThis);
        // The balance of the messenger contract is unchanged.
        assertEq(address(l1CrossDomainMessenger).balance, balanceBeforeMessenger);
    }
}
