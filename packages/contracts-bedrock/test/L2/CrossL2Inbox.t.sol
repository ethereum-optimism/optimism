// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { VmSafe } from "forge-std/Vm.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";
import { IL1EventRegistry } from "interfaces/L1/IL1EventRegistry.sol";
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { IL2ToL1MessagePasser } from "interfaces/L2/IL2ToL1MessagePasser.sol";
import { ILocalLogOracle } from "interfaces/L2/ILocalLogOracle.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { ProjectionEventExporter } from "src/private-interop/ProjectionEventExporter.sol";
import { AttestedEventVerifier } from "src/private-interop/AttestedEventVerifier.sol";

/// @title CrossL2Inbox_ValidateMessageRelayer_Harness
/// @notice For test contract used to validate multiple messages in a single tx.
contract CrossL2Inbox_ValidateMessageRelayer_Harness is Test {
    ICrossL2Inbox public immutable CROSS_L2_INBOX;

    constructor(address _crossL2Inbox) {
        CROSS_L2_INBOX = ICrossL2Inbox(_crossL2Inbox);
    }

    /// @notice Validates a message and retries it after it reverts.
    function validateAndRetry(Identifier memory _id, bytes32 _messageHash) external {
        try CROSS_L2_INBOX.validateMessage(_id, _messageHash) {
            // It should always revert
            assertFalse(true);
        } catch {
            // It should revert with NotInAccessList when called a second time without any access
            // list
            vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
            CROSS_L2_INBOX.validateMessage(_id, _messageHash);
        }
    }

    /// @notice Validates multiple messages in a single tx.
    function validateMessages(Identifier[20] memory _ids, bytes32[20] memory _messageHashes) external {
        for (uint256 i; i < _ids.length; i++) {
            CROSS_L2_INBOX.validateMessage(_ids[i], _messageHashes[i]);
        }
    }
}

/// @title CrossL2Inbox_Test_Init
/// @notice Reusable test initialization for `CrossL2Inbox` tests.
abstract contract CrossL2Inbox_TestInit is CommonTest {
    event ExecutingMessage(bytes32 indexed msgHash, Identifier id);
    event ExecutingCertifiedMessage(bytes32 indexed msgHash, Identifier id);
    event EventExported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);
    event EventImported(bytes32 indexed checksum, bytes32 indexed payloadHash, Identifier id);

    CrossL2Inbox_ValidateMessageRelayer_Harness public validateMessageRelayer;

    mapping(bytes32 => bool) public relayedMessages;
    mapping(bytes32 => bool) public warmedSlots;

    function setUp() public virtual override {
        useInteropOverride = true;
        super.setUp();
        validateMessageRelayer = new CrossL2Inbox_ValidateMessageRelayer_Harness(address(crossL2Inbox));
    }
}

/// @title CrossL2Inbox_CertifiedEvent_TestInit
/// @notice Tests exporting and importing event certificates through L1.
abstract contract CrossL2Inbox_CertifiedEvent_TestInit is CrossL2Inbox_TestInit {
    address internal l1EventRegistry = makeAddr("l1EventRegistry");
    Identifier internal id;
    bytes32 internal payloadHash = keccak256("payload");

    function setUp() public virtual override {
        super.setUp();

        vm.prank(IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        crossL2Inbox.setL1EventRegistry(l1EventRegistry);

        vm.roll(100);
        vm.warp(1_000_000);
        id = Identifier({
            origin: makeAddr("origin"),
            blockNumber: block.number - 1,
            logIndex: 2,
            timestamp: block.timestamp,
            chainId: block.chainid
        });
    }
}

/// @title CrossL2Inbox_ExportEvent_Test
/// @notice Tests the `exportEvent` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_ExportEvent_Test is CrossL2Inbox_CertifiedEvent_TestInit {
    function test_exportEvent_succeeds() external {
        // The exact seven-day boundary remains eligible.
        id.timestamp = block.timestamp - crossL2Inbox.EVENT_LOOKUP_WINDOW();
        bytes memory oracleCall = abi.encodeCall(ILocalLogOracle.containsLog, (id, payloadHash));
        vm.mockCall(Predeploys.LOCAL_LOG_ORACLE, oracleCall, abi.encode(true));

        bytes memory registryCall = abi.encodeCall(IL1EventRegistry.registerEvent, (id, payloadHash));
        ICrossDomainMessenger messenger = ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        IL2ToL1MessagePasser passer = IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER));
        uint256 withdrawalNonce = passer.messageNonce();
        bytes memory relayCall = abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (
                messenger.messageNonce(),
                Predeploys.CROSS_L2_INBOX,
                l1EventRegistry,
                0,
                uint256(crossL2Inbox.REGISTER_EVENT_GAS_LIMIT()),
                registryCall
            )
        );
        bytes32 withdrawalHash = Hashing.hashWithdrawal(
            Types.WithdrawalTransaction({
                nonce: withdrawalNonce,
                sender: Predeploys.L2_CROSS_DOMAIN_MESSENGER,
                target: address(messenger.otherMessenger()),
                value: 0,
                gasLimit: messenger.baseGas(registryCall, crossL2Inbox.REGISTER_EVENT_GAS_LIMIT()),
                data: relayCall
            })
        );
        vm.expectCall(Predeploys.LOCAL_LOG_ORACLE, oracleCall);

        bytes32 checksum = crossL2Inbox.calculateChecksum(id, payloadHash);
        vm.expectEmit(address(crossL2Inbox));
        emit EventExported(checksum, payloadHash, id);
        crossL2Inbox.exportEvent(id, payloadHash);
        assertTrue(passer.sentMessages(withdrawalHash));
        assertEq(passer.messageNonce(), withdrawalNonce + 1);
        // This is the exact slot the standard portal proves, not merely an emitted log.
        assertEq(vm.load(address(passer), keccak256(abi.encode(withdrawalHash, uint256(0)))), bytes32(uint256(1)));
    }

    function test_exportEvent_tooOld_reverts() external {
        id.timestamp = block.timestamp - crossL2Inbox.EVENT_LOOKUP_WINDOW() - 1;

        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventTooOld.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_oracleRejects_reverts() external {
        vm.mockCall(
            Predeploys.LOCAL_LOG_ORACLE,
            abi.encodeCall(ILocalLogOracle.containsLog, (id, payloadHash)),
            abi.encode(false)
        );
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventNotFound.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_oracleReverts_reverts() external {
        vm.mockCallRevert(
            Predeploys.LOCAL_LOG_ORACLE,
            abi.encodeCall(ILocalLogOracle.containsLog, (id, payloadHash)),
            bytes("unavailable")
        );
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventNotFound.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_oracleMissing_reverts() external {
        vm.etch(Predeploys.LOCAL_LOG_ORACLE, bytes(""));
        vm.expectRevert();
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_wrongChain_reverts() external {
        id.chainId++;
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventFromAnotherChain.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_currentBlock_reverts() external {
        id.blockNumber = block.number;
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventNotInPreviousBlock.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }

    function test_exportEvent_futureTimestamp_reverts() external {
        id.timestamp = block.timestamp + 1;
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventNotInPreviousBlock.selector);
        crossL2Inbox.exportEvent(id, payloadHash);
    }
}

/// @title CrossL2Inbox_ImportEvent_Test
/// @notice Tests the `importEvent` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_ImportEvent_Test is CrossL2Inbox_CertifiedEvent_TestInit {
    function test_importEvent_succeeds() external {
        bytes32 checksum = crossL2Inbox.calculateChecksum(id, payloadHash);

        vm.expectEmit(address(crossL2Inbox));
        emit EventImported(checksum, payloadHash, id);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1EventRegistry));
        crossL2Inbox.importEvent(id, payloadHash);

        assertTrue(crossL2Inbox.certifiedMessages(checksum));
    }

    function test_importEvent_untrustedSender_reverts() external {
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_NotEventRegistry.selector);
        crossL2Inbox.importEvent(id, payloadHash);
    }

    function test_importEvent_unaliasedRegistry_reverts() external {
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_NotEventRegistry.selector);
        vm.prank(l1EventRegistry);
        crossL2Inbox.importEvent(id, payloadHash);
    }
}

/// @notice Tests governance control over the pluggable proof policy.
contract CrossL2Inbox_SetEventProofVerifier_Test is CrossL2Inbox_TestInit {
    function test_setEventProofVerifier_unauthorized_reverts() external {
        vm.prank(makeAddr("unauthorized"));
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        crossL2Inbox.setEventProofVerifier(address(0));
    }

    function test_setEventProofVerifier_noCode_reverts() external {
        vm.prank(IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_InvalidEventProofVerifier.selector);
        crossL2Inbox.setEventProofVerifier(makeAddr("noCode"));
    }

    function test_setEventProofVerifier_disable_succeeds() external {
        AttestedEventVerifier verifier = new AttestedEventVerifier(makeAddr("signer"), address(crossL2Inbox));
        vm.startPrank(IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        crossL2Inbox.setEventProofVerifier(address(verifier));
        crossL2Inbox.setEventProofVerifier(address(0));
        vm.stopPrank();
        assertEq(crossL2Inbox.eventProofVerifier(), address(0));
    }
}

/// @notice Tests pluggable proof exports, persistent consumption and rollback on export failure.
contract CrossL2Inbox_ExportProvenEvent_Test is CrossL2Inbox_CertifiedEvent_TestInit {
    uint256 internal signerKey = 123;
    AttestedEventVerifier internal verifier;

    function setUp() public virtual override {
        super.setUp();
        vm.etch(Predeploys.PROJECTION_EVENT_EXPORTER, address(new ProjectionEventExporter()).code);
        verifier = new AttestedEventVerifier(vm.addr(signerKey), address(crossL2Inbox));
        _configure(verifier);
    }

    function _configure(AttestedEventVerifier _verifier) internal {
        vm.prank(IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner());
        crossL2Inbox.setEventProofVerifier(address(_verifier));
    }

    function _proof() internal view returns (bytes memory) {
        (uint8 v, bytes32 r, bytes32 s) = vm.sign(signerKey, verifier.eventDigest(id, payloadHash));
        return abi.encodePacked(r, s, v);
    }

    function test_exportProvenEvent_privateProxyRevertsProjectionProxyExports_succeeds() external {
        bytes32 implementationSlot = bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1);
        address exporter = Predeploys.PROJECTION_EVENT_EXPORTER;
        vm.etch(exporter, address(new Proxy(Predeploys.PROXY_ADMIN)).code);
        vm.store(exporter, implementationSlot, bytes32(0));
        bytes memory proof = _proof();
        uint256 nonce = IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).messageNonce();
        vm.expectRevert("Proxy: implementation not initialized");
        ProjectionEventExporter(exporter).exportProvenEvent(id, payloadHash, proof);
        assertFalse(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
        assertFalse(verifier.consumedEvents(keccak256(abi.encode(id))));
        assertEq(IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).messageNonce(), nonce);
        vm.store(exporter, implementationSlot, bytes32(uint256(uint160(address(new ProjectionEventExporter())))));
        ProjectionEventExporter(exporter).exportProvenEvent(id, payloadHash, proof);
        assertTrue(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
        assertEq(IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).messageNonce(), nonce + 1);
    }

    function test_exportProvenEvent_directInboxCall_reverts() external {
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_NotProjectionEventExporter.selector);
        crossL2Inbox.exportProvenEvent(id, payloadHash, bytes(""));
    }

    function test_exportProvenEvent_withoutLocalOracle_succeeds() external {
        id.timestamp = block.timestamp - 8 days;
        vm.etch(Predeploys.LOCAL_LOG_ORACLE, bytes(""));
        uint256 nonce = IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).messageNonce();
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, payloadHash, _proof());
        assertTrue(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
        assertEq(IL2ToL1MessagePasser(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).messageNonce(), nonce + 1);
    }

    function test_exportProvenEvent_changedPayload_reverts() external {
        bytes memory proof = _proof();
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_InvalidEventProof.selector);
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, bytes32(uint256(1)), proof);
        assertFalse(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
    }

    function test_exportProvenEvent_afterVerifierReplacement_reverts() external {
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, payloadHash, _proof());
        verifier = new AttestedEventVerifier(vm.addr(signerKey), address(crossL2Inbox));
        _configure(verifier);
        payloadHash = keccak256("conflicting event");
        bytes memory proof = _proof();
        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_EventAlreadyExported.selector);
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, payloadHash, proof);
    }

    function test_exportProvenEvent_exportFailureDoesNotConsume_succeeds() external {
        bytes memory proof = _proof();
        vm.mockCallRevert(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            abi.encodeWithSelector(ICrossDomainMessenger.sendMessage.selector),
            bytes("messenger unavailable")
        );
        vm.expectRevert(bytes("messenger unavailable"));
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, payloadHash, proof);
        assertFalse(verifier.consumedEvents(keccak256(abi.encode(id))));
        assertFalse(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
        vm.clearMockedCalls();
        ProjectionEventExporter(Predeploys.PROJECTION_EVENT_EXPORTER).exportProvenEvent(id, payloadHash, proof);
        assertTrue(crossL2Inbox.provenEvents(keccak256(abi.encode(id))));
    }
}

/// @title CrossL2Inbox_CertificateReceiver_Harness
/// @notice Recipient used to test atomic certificate import and retry after application failure.
contract CrossL2Inbox_CertificateReceiver_Harness {
    error ApplicationFailure();

    bool public shouldFail = true;
    uint256 public calls;

    function allowCalls() external {
        shouldFail = false;
    }

    function receiveMessage() external {
        if (shouldFail) revert ApplicationFailure();
        calls++;
    }
}

/// @title CrossL2Inbox_ImportAndExecute_Test
/// @notice Tests real messenger execution, rollback and duplicate protection for certified deposits.
contract CrossL2Inbox_ImportAndExecute_Test is CrossL2Inbox_CertifiedEvent_TestInit {
    function test_importAndExecute_retryAfterApplicationFailure_succeeds() external {
        CrossL2Inbox_CertificateReceiver_Harness receiver = new CrossL2Inbox_CertificateReceiver_Harness();
        id.origin = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
        id.chainId = block.chainid + 1;
        bytes memory callData = abi.encodeCall(receiver.receiveMessage, ());
        address sender = makeAddr("privateSender");
        bytes memory sentMessage = bytes.concat(
            abi.encode(
                keccak256("SentMessage(uint256,address,uint256,address,bytes)"),
                block.chainid,
                address(receiver),
                uint256(0)
            ),
            abi.encode(sender, callData)
        );
        bytes32 checksum = crossL2Inbox.calculateChecksum(id, keccak256(sentMessage));
        bytes32 messageHash =
            Hashing.hashL2toL2CrossDomainMessage(block.chainid, id.chainId, 0, sender, address(receiver), callData);
        IL2ToL2CrossDomainMessenger messenger = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);

        vm.fee(1);
        vm.txGasPrice(0);
        vm.expectRevert(CrossL2Inbox_CertificateReceiver_Harness.ApplicationFailure.selector);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1EventRegistry));
        crossL2Inbox.importAndExecute(id, sentMessage);
        assertFalse(crossL2Inbox.certifiedMessages(checksum));
        assertFalse(messenger.successfulMessages(messageHash));
        assertEq(receiver.calls(), 0);

        receiver.allowCalls();
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1EventRegistry));
        crossL2Inbox.importAndExecute(id, sentMessage);
        assertTrue(crossL2Inbox.certifiedMessages(checksum));
        assertTrue(messenger.successfulMessages(messageHash));
        assertEq(receiver.calls(), 1);

        vm.expectRevert(IL2ToL2CrossDomainMessenger.MessageAlreadyRelayed.selector);
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1EventRegistry));
        crossL2Inbox.importAndExecute(id, sentMessage);
        assertEq(receiver.calls(), 1);
    }
}

/// @title CrossL2Inbox_ValidateMessage_Test
/// @notice Tests the `validateMessage` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_ValidateMessage_Test is CrossL2Inbox_CertifiedEvent_TestInit {
    function test_validateMessage_certifiedDeposit_succeeds() external {
        vm.prank(AddressAliasHelper.applyL1ToL2Alias(l1EventRegistry));
        crossL2Inbox.importEvent(id, payloadHash);

        vm.fee(1);
        vm.txGasPrice(0);
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingCertifiedMessage(payloadHash, id);
        crossL2Inbox.validateMessage(id, payloadHash);
    }

    /// @notice Test that `validateMessage` reverts when executed in a deposit transaction.
    function testFuzz_validateMessage_depositTransaction_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        vm.expectRevert(ICrossL2Inbox.CrossL2Inbox_NoExecutingDeposits.selector);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that `validateMessage` succeeds with a zero gas price when the base fee is
    ///         also zero. Chains configured with a zero base fee may execute user transactions
    ///         with `tx.gasprice == 0`, so the deposit check must not reject them.
    /// forge-config: default.isolate = true
    function testFuzz_validateMessage_zeroBaseFeeZeroGasPrice_succeeds(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Zero base fee chain with a zero gas price transaction
        vm.fee(0);
        vm.txGasPrice(0);

        // Prepare the access list to be sent with the next call
        bytes32 slot = crossL2Inbox.calculateChecksum(_id, _messageHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slot;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` event to be emitted
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingMessage(_messageHash, _id);

        // Validate the message
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that `validateMessage` reverts when the slot is not warm.
    function testFuzz_validateMessage_accessList_reverts(Identifier memory _id, bytes32 _messageHash) external {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Cold all the slots
        vm.cool(address(crossL2Inbox));

        // Expect revert
        vm.txGasPrice(block.basefee);
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that `validateMessage` succeeds when the slot for the message checksum is
    ///         warm.
    /// forge-config: default.isolate = true
    function testFuzz_validateMessage_succeeds(
        Identifier memory _id,
        bytes32 _messageHash
    )
        public
        returns (bytes32 slot_)
    {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Prepare the access list to be sent with the next call
        slot_ = crossL2Inbox.calculateChecksum(_id, _messageHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slot_;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` event to be emitted
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingMessage(_messageHash, _id);

        // Validate the message
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that multiple calls to `validateMessage` with different access lists don't
    ///         collide and succeeds.
    /// @dev This tests that the way we encode and hash the checksum slot is unique enough to avoid
    ///      collisions.
    /// forge-config: default.isolate = true
    function testFuzz_validateMessage_multipleAccessLists_succeeds(
        Identifier[20] memory _ids,
        bytes32[20] calldata _messageHash
    )
        external
    {
        // Send batches of calls with different access lists and check they never collide and
        // always succeed
        for (uint256 i; i < _ids.length; i++) {
            // Make sure we're not re-validating the same message
            bytes32 msgToValidate = keccak256(abi.encode(_ids[i], _messageHash[i]));
            vm.assume(relayedMessages[msgToValidate] == false);
            relayedMessages[msgToValidate] = true;

            // Call validateMessage and get the slot
            bytes32 slot_ = testFuzz_validateMessage_succeeds(_ids[i], _messageHash[i]);
            // Check that the slot doesn't match a previously warmed slot
            assertEq(warmedSlots[slot_], false);
            // Mark the slot as warmed
            warmedSlots[slot_] = true;

            // Remove the access list
            vm.noAccessList();
        }
    }

    /// @notice Test that an invalid tx calling `validateMessage` doesn't warm the slot for the
    ///         next one.
    /// forge-config: default.isolate = true
    function test_validateMessage_revertDoesntWarm_reverts(
        Identifier memory _idOne,
        Identifier memory _idTwo,
        bytes32 _messageHashOne,
        bytes32 _messageHashTwo
    )
        external
    {
        // Bound values types to ensure they are not too large
        _idOne.blockNumber = bound(_idOne.blockNumber, 0, type(uint64).max);
        _idOne.logIndex = bound(_idOne.logIndex, 0, type(uint32).max);
        _idOne.timestamp = bound(_idOne.timestamp, 0, type(uint64).max);
        _idTwo.blockNumber = bound(_idTwo.blockNumber, 0, type(uint64).max);
        _idTwo.logIndex = bound(_idTwo.logIndex, 0, type(uint32).max);
        _idTwo.timestamp = bound(_idTwo.timestamp, 0, type(uint64).max);

        // Make sure the first message is valid
        bytes32 slotTwo = crossL2Inbox.calculateChecksum(_idTwo, _messageHashTwo);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slotTwo;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect a revert on the tx1 warming the slot two
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_idOne, _messageHashOne);

        // Send the tx2 but without any access list and check that it reverts since the slot should not be warmed
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_idTwo, _messageHashTwo);
    }

    /// @notice Test that a valid tx calling `validateMessage` doesn't warm the slot for the next
    ///         one.
    /// forge-config: default.isolate = true
    function test_validateMessage_validDoesntWarm_reverts(Identifier memory _id, bytes32 _messageHash) external {
        // Bound values types to ensure they are not too large
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Make sure the first message is valid
        bytes32 slotOne = crossL2Inbox.calculateChecksum(_id, _messageHash);
        bytes32[] memory slots = new bytes32[](1);
        slots[0] = slotOne;
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` event to be emitted
        vm.expectEmit(address(crossL2Inbox));
        emit ExecutingMessage(_messageHash, _id);

        // Validate the message
        vm.accessList(accessList);
        crossL2Inbox.validateMessage(_id, _messageHash);

        // Send the same msg but without any access list and check that it reverts since the
        // slot should not be warmed
        vm.expectRevert(ICrossL2Inbox.NotInAccessList.selector);
        crossL2Inbox.validateMessage(_id, _messageHash);
    }

    /// @notice Test that an invalid message without access list does not warm the slot and
    ///         fails the second time.
    function test_validateMessage_sameMsgWithoutAccessListTwice_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        public
    {
        // Make sure the Identifier is valid
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);
        _id.timestamp = bound(_id.timestamp, 0, type(uint64).max);

        // Try and retry the message without any access list
        vm.txGasPrice(block.basefee);
        vm.expectCall(address(crossL2Inbox), abi.encodeCall(ICrossL2Inbox.validateMessage, (_id, _messageHash)), 2);
        validateMessageRelayer.validateAndRetry(_id, _messageHash);
    }

    /// @notice Test that multiple calls to `validateMessage` with multiple storage keys succeeds
    ///         on the same tx.
    /// forge-config: default.isolate = true
    function test_validateMessage_multipleStorageKeys_succeeds(
        Identifier[20] memory _ids,
        bytes32[20] memory _messageHashes
    )
        public
    {
        bytes32[] memory slots = new bytes32[](_ids.length);
        for (uint256 i; i < _ids.length; i++) {
            // Make sure the Identifier is valid
            _ids[i].blockNumber = bound(_ids[i].blockNumber, 0, type(uint64).max);
            _ids[i].logIndex = bound(_ids[i].logIndex, 0, type(uint32).max);
            _ids[i].timestamp = bound(_ids[i].timestamp, 0, type(uint64).max);

            // Calculate the checksum for the message and add it to the storage keys
            bytes32 slot = crossL2Inbox.calculateChecksum(_ids[i], _messageHashes[i]);
            slots[i] = slot;
        }

        // Prepare the access list to be sent with the next txs
        VmSafe.AccessListItem[] memory accessList = new VmSafe.AccessListItem[](1);
        accessList[0] = VmSafe.AccessListItem({ target: address(crossL2Inbox), storageKeys: slots });

        // Expect `ExecutingMessage` events to be emitted
        for (uint256 i; i < _ids.length; i++) {
            vm.expectEmit(address(crossL2Inbox));
            emit ExecutingMessage(_messageHashes[i], _ids[i]);
        }

        // Validate the message
        vm.accessList(accessList);
        validateMessageRelayer.validateMessages(_ids, _messageHashes);
    }
}

/// @title CrossL2Inbox_CalculateChecksum_Test
/// @notice Tests the `calculateChecksum` function of the `CrossL2Inbox` contract.
contract CrossL2Inbox_CalculateChecksum_Test is CrossL2Inbox_TestInit {
    /// @notice Test that `calculateChecksum` reverts when the block number is greater than 2^64.
    function testFuzz_calculateChecksum_withTooLargeBlockNumber_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        // Set to the 2**64 + 1
        _id.blockNumber = 18446744073709551615 + 1;
        vm.expectRevert(ICrossL2Inbox.BlockNumberTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` reverts when the log index is greater than 2^32.
    function testFuzz_calculateChecksum_withTooLargeLogIndex_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);

        // Set to the 2**32 + 1
        _id.logIndex = 4294967295 + 1;
        vm.expectRevert(ICrossL2Inbox.LogIndexTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` reverts when the timestamp is greater than 2^64.
    function testFuzz_calculateChecksum_withTooLargeTimestamp_reverts(
        Identifier memory _id,
        bytes32 _messageHash
    )
        external
    {
        _id.blockNumber = bound(_id.blockNumber, 0, type(uint64).max);
        _id.logIndex = bound(_id.logIndex, 0, type(uint32).max);

        // Set to the 2**64 + 1
        _id.timestamp = 18446744073709551615 + 1;
        vm.expectRevert(ICrossL2Inbox.TimestampTooHigh.selector);
        crossL2Inbox.calculateChecksum(_id, _messageHash);
    }

    /// @notice Test that `calculateChecksum` succeeds matching the expected calculated checksum.
    /// @dev Using a hardcoded checksum manually calculated and verified.
    function test_calculateChecksum_succeeds() external view {
        Identifier memory id = Identifier(
            address(0),
            uint64(0xa1a2a3a4a5a6a7a8),
            uint32(0xb1b2b3b4),
            uint64(0xc1c2c3c4c5c6c7c8),
            uint256(0xd1d2d3d4d5d6d7d8)
        );

        // Calculate the expected checksum.
        bytes32 messageHash = 0x8017559a85b12c04b14a1a425d53486d1015f833714a09bd62f04152a7e2ae9b;
        bytes32 checksum = crossL2Inbox.calculateChecksum(id, messageHash);
        bytes32 expectedChecksum = 0x03139ddd21106abad4bb82800fedfa3a103f53f242c2d5b7615b0baad8379531;

        // Expect it to match
        assertEq(checksum, expectedChecksum);
    }
}
