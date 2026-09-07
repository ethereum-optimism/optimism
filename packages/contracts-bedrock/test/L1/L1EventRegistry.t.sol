// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { L1EventRegistry } from "src/L1/L1EventRegistry.sol";
import { L1CrossDomainMessenger } from "src/L1/L1CrossDomainMessenger.sol";
import { Proxy } from "src/universal/Proxy.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Hashing } from "src/libraries/Hashing.sol";
import { Encoding } from "src/libraries/Encoding.sol";

// Interfaces
import { IL1EventRegistry } from "interfaces/L1/IL1EventRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IL1CrossDomainMessenger } from "interfaces/L1/IL1CrossDomainMessenger.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title L1EventRegistry_TestInit
/// @notice Reusable test initialization for L1EventRegistry tests.
abstract contract L1EventRegistry_TestInit is Test {
    event EventRegistered(bytes32 indexed certificate, bytes32 indexed payloadHash, Identifier id);

    IETHLockbox internal lockbox;
    IOptimismPortal2 internal sourcePortal;
    IOptimismPortal2 internal destinationPortal;
    ISystemConfig internal sourceSystemConfig;
    IL1CrossDomainMessenger internal sourceMessenger;
    L1EventRegistry internal registry;

    Identifier internal id;
    bytes32 internal payloadHash = keccak256("payload");

    function setUp() public virtual {
        lockbox = IETHLockbox(makeAddr("lockbox"));
        sourcePortal = IOptimismPortal2(payable(makeAddr("sourcePortal")));
        destinationPortal = IOptimismPortal2(payable(makeAddr("destinationPortal")));
        sourceSystemConfig = ISystemConfig(makeAddr("sourceSystemConfig"));
        sourceMessenger = IL1CrossDomainMessenger(makeAddr("sourceMessenger"));
        registry = new L1EventRegistry(lockbox);

        id = Identifier({ origin: makeAddr("origin"), blockNumber: 100, logIndex: 2, timestamp: 1_000, chainId: 901 });

        _mockAuthorizedPortal(sourcePortal);
        _mockAuthorizedPortal(destinationPortal);
        vm.mockCall(
            address(sourceMessenger),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(Predeploys.CROSS_L2_INBOX)
        );
        vm.mockCall(
            address(sourceMessenger), abi.encodeCall(IL1CrossDomainMessenger.portal, ()), abi.encode(sourcePortal)
        );
        vm.mockCall(
            address(sourcePortal), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(sourceSystemConfig)
        );
        vm.mockCall(address(sourceSystemConfig), abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(id.chainId));
        vm.mockCall(
            address(sourceSystemConfig),
            abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()),
            abi.encode(sourceMessenger)
        );
    }

    function _mockAuthorizedPortal(IOptimismPortal2 _portal) internal {
        vm.mockCall(address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (_portal)), abi.encode(true));
        vm.mockCall(address(_portal), abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(lockbox));
    }

    function _registerEvent() internal {
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);
    }
}

/// @title L1EventRegistry_RegisterEvent_Test
/// @notice Tests event registration through a source portal.
contract L1EventRegistry_RegisterEvent_Test is L1EventRegistry_TestInit {
    function test_registerEvent_forgedMessenger_reverts() external {
        address forged = makeAddr("forgedMessenger");
        // Claiming a real portal and inbox sender is insufficient: SystemConfig binds the caller.
        vm.mockCall(forged, abi.encodeCall(IL1CrossDomainMessenger.portal, ()), abi.encode(sourcePortal));
        vm.mockCall(
            forged,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(Predeploys.CROSS_L2_INBOX)
        );
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedMessenger.selector);
        vm.prank(forged);
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_portalLockboxMismatch_reverts() external {
        vm.mockCall(address(sourcePortal), abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(0xbad)));
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedPortal.selector);
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_succeeds() external {
        bytes32 certificate = registry.calculateCertificate(id, payloadHash);

        vm.expectEmit(address(registry));
        emit EventRegistered(certificate, payloadHash, id);
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);

        assertTrue(registry.registeredEvents(certificate));
    }

    function test_registerEvent_wrongL2Sender_reverts() external {
        vm.mockCall(
            address(sourceMessenger),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(0xbad))
        );

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedL2Sender.selector);
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_wrongSourceChain_reverts() external {
        vm.mockCall(
            address(sourceSystemConfig), abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(id.chainId + 1)
        );

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_WrongSourceChain.selector);
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_unauthorizedPortal_reverts() external {
        vm.mockCall(address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (sourcePortal)), abi.encode(false));

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedPortal.selector);
        vm.prank(address(sourceMessenger));
        registry.registerEvent(id, payloadHash);
    }
}

/// @title L1EventRegistry_RelayEvent_Test
/// @notice Tests relaying finalized event certificates into a destination portal.
contract L1EventRegistry_RelayEvent_Test is L1EventRegistry_TestInit {
    function test_relayEvent_unauthorizedDestination_reverts() external {
        _registerEvent();
        vm.mockCall(
            address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (destinationPortal)), abi.encode(false)
        );
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedPortal.selector);
        registry.relayEvent(destinationPortal, id, payloadHash, 250_000);
    }

    function test_relayEvent_destinationLockboxMismatch_reverts() external {
        _registerEvent();
        vm.mockCall(
            address(destinationPortal), abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(address(0xbad))
        );
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedPortal.selector);
        registry.relayEvent(destinationPortal, id, payloadHash, 250_000);
    }

    function test_relayEvent_alteredIdentifier_reverts() external {
        _registerEvent();
        id.logIndex++;
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_EventNotRegistered.selector);
        registry.relayEvent(destinationPortal, id, payloadHash, 250_000);
    }

    function test_relayMessage_alteredPayload_reverts() external {
        _registerEvent();
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_EventNotRegistered.selector);
        registry.relayMessage(destinationPortal, id, bytes("different payload"), 250_000);
    }

    function test_relayEvent_succeeds() external {
        _registerEvent();
        uint64 gasLimit = 250_000;
        bytes memory data = abi.encodeCall(ICrossL2Inbox.importEvent, (id, payloadHash));

        vm.expectCall(
            address(destinationPortal),
            abi.encodeCall(IOptimismPortal2.depositTransaction, (Predeploys.CROSS_L2_INBOX, 0, gasLimit, false, data))
        );
        registry.relayEvent(destinationPortal, id, payloadHash, gasLimit);
    }

    function test_relayEvent_unregistered_reverts() external {
        vm.expectRevert(IL1EventRegistry.L1EventRegistry_EventNotRegistered.selector);
        registry.relayEvent(destinationPortal, id, payloadHash, 250_000);
    }

    function test_relayMessage_succeeds() external {
        bytes memory sentMessage = hex"deadbeef";
        payloadHash = keccak256(sentMessage);
        _registerEvent();
        uint64 gasLimit = 500_000;
        bytes memory data = abi.encodeCall(ICrossL2Inbox.importAndExecute, (id, sentMessage));

        vm.expectCall(
            address(destinationPortal),
            abi.encodeCall(IOptimismPortal2.depositTransaction, (Predeploys.CROSS_L2_INBOX, 0, gasLimit, false, data))
        );
        registry.relayMessage(destinationPortal, id, sentMessage, gasLimit);
    }
}

/// @title L1EventRegistry_Integration_Test
/// @notice Exercises delivery and retry through a real L1 messenger. Portal finality is an
///         upstream boundary, modeled here by its authenticated l2Sender context.
contract L1EventRegistry_Integration_Test is L1EventRegistry_TestInit {
    uint256 internal nonce;
    bytes internal message;
    bytes32 internal messageHash;

    function setUp() public override {
        super.setUp();
        Proxy proxy = new Proxy(address(this));
        proxy.upgradeTo(address(new L1CrossDomainMessenger()));
        sourceMessenger = IL1CrossDomainMessenger(address(proxy));
        sourceMessenger.initialize(sourceSystemConfig, sourcePortal);
        vm.mockCall(address(sourceSystemConfig), abi.encodeCall(ISystemConfig.paused, ()), abi.encode(false));
        vm.mockCall(
            address(sourceSystemConfig),
            abi.encodeCall(ISystemConfig.l1CrossDomainMessenger, ()),
            abi.encode(sourceMessenger)
        );
        vm.mockCall(
            address(sourcePortal),
            abi.encodeCall(IOptimismPortal2.l2Sender, ()),
            abi.encode(Predeploys.L2_CROSS_DOMAIN_MESSENGER)
        );
        nonce = Encoding.encodeVersionedNonce(0, 1);
        message = abi.encodeCall(IL1EventRegistry.registerEvent, (id, payloadHash));
        messageHash =
            Hashing.hashCrossDomainMessageV1(nonce, Predeploys.CROSS_L2_INBOX, address(registry), 0, 200_000, message);
    }

    function _deliver() internal {
        sourceMessenger.relayMessage(nonce, Predeploys.CROSS_L2_INBOX, address(registry), 0, 200_000, message);
    }

    function test_registerEvent_messengerDelivery_succeeds() external {
        vm.prank(address(sourcePortal));
        _deliver();
        assertTrue(registry.registeredEvents(registry.calculateCertificate(id, payloadHash)));
        assertTrue(sourceMessenger.successfulMessages(messageHash));
    }

    function test_registerEvent_retryAfterFailedDelivery_succeeds() external {
        // A withdrawal may finalize while cluster membership temporarily prevents registration.
        vm.mockCall(address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (sourcePortal)), abi.encode(false));
        vm.prank(address(sourcePortal));
        _deliver();
        assertFalse(registry.registeredEvents(registry.calculateCertificate(id, payloadHash)));
        assertTrue(sourceMessenger.failedMessages(messageHash));
        assertFalse(sourceMessenger.successfulMessages(messageHash));

        // The original lookup window may expire before recovery. Retrying needs no new L2 export.
        vm.warp(block.timestamp + 8 days);
        _mockAuthorizedPortal(sourcePortal);
        _deliver();
        assertTrue(registry.registeredEvents(registry.calculateCertificate(id, payloadHash)));
        assertTrue(sourceMessenger.successfulMessages(messageHash));
    }

    function test_registerEvent_unprovenMessengerDelivery_reverts() external {
        vm.expectRevert("CrossDomainMessenger: message cannot be replayed");
        _deliver();
        assertFalse(registry.registeredEvents(registry.calculateCertificate(id, payloadHash)));
    }
}
