// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";

// Contracts
import { L1EventRegistry } from "src/L1/L1EventRegistry.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IL1EventRegistry } from "interfaces/L1/IL1EventRegistry.sol";
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { ICrossL2Inbox, Identifier } from "interfaces/L2/ICrossL2Inbox.sol";

/// @title L1EventRegistry_TestInit
/// @notice Reusable test initialization for L1EventRegistry tests.
abstract contract L1EventRegistry_TestInit is Test {
    event EventRegistered(bytes32 indexed certificate, bytes32 indexed payloadHash, Identifier id);

    IETHLockbox internal lockbox;
    IOptimismPortal2 internal sourcePortal;
    IOptimismPortal2 internal destinationPortal;
    ISystemConfig internal sourceSystemConfig;
    L1EventRegistry internal registry;

    Identifier internal id;
    bytes32 internal payloadHash = keccak256("payload");

    function setUp() public virtual {
        lockbox = IETHLockbox(makeAddr("lockbox"));
        sourcePortal = IOptimismPortal2(payable(makeAddr("sourcePortal")));
        destinationPortal = IOptimismPortal2(payable(makeAddr("destinationPortal")));
        sourceSystemConfig = ISystemConfig(makeAddr("sourceSystemConfig"));
        registry = new L1EventRegistry(lockbox);

        id = Identifier({ origin: makeAddr("origin"), blockNumber: 100, logIndex: 2, timestamp: 1_000, chainId: 901 });

        _mockAuthorizedPortal(sourcePortal);
        _mockAuthorizedPortal(destinationPortal);
        vm.mockCall(
            address(sourcePortal), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(Predeploys.CROSS_L2_INBOX)
        );
        vm.mockCall(
            address(sourcePortal), abi.encodeCall(IOptimismPortal2.systemConfig, ()), abi.encode(sourceSystemConfig)
        );
        vm.mockCall(address(sourceSystemConfig), abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(id.chainId));
    }

    function _mockAuthorizedPortal(IOptimismPortal2 _portal) internal {
        vm.mockCall(address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (_portal)), abi.encode(true));
        vm.mockCall(address(_portal), abi.encodeCall(IOptimismPortal2.ethLockbox, ()), abi.encode(lockbox));
    }

    function _registerEvent() internal {
        vm.prank(address(sourcePortal));
        registry.registerEvent(id, payloadHash);
    }
}

/// @title L1EventRegistry_RegisterEvent_Test
/// @notice Tests event registration through a source portal.
contract L1EventRegistry_RegisterEvent_Test is L1EventRegistry_TestInit {
    function test_registerEvent_succeeds() external {
        bytes32 certificate = registry.calculateCertificate(id, payloadHash);

        vm.expectEmit(address(registry));
        emit EventRegistered(certificate, payloadHash, id);
        vm.prank(address(sourcePortal));
        registry.registerEvent(id, payloadHash);

        assertTrue(registry.registeredEvents(certificate));
    }

    function test_registerEvent_wrongL2Sender_reverts() external {
        vm.mockCall(address(sourcePortal), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(address(0xbad)));

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedL2Sender.selector);
        vm.prank(address(sourcePortal));
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_wrongSourceChain_reverts() external {
        vm.mockCall(
            address(sourceSystemConfig), abi.encodeCall(ISystemConfig.l2ChainId, ()), abi.encode(id.chainId + 1)
        );

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_WrongSourceChain.selector);
        vm.prank(address(sourcePortal));
        registry.registerEvent(id, payloadHash);
    }

    function test_registerEvent_unauthorizedPortal_reverts() external {
        vm.mockCall(address(lockbox), abi.encodeCall(IETHLockbox.authorizedPortals, (sourcePortal)), abi.encode(false));

        vm.expectRevert(IL1EventRegistry.L1EventRegistry_UnauthorizedPortal.selector);
        vm.prank(address(sourcePortal));
        registry.registerEvent(id, payloadHash);
    }
}

/// @title L1EventRegistry_RelayEvent_Test
/// @notice Tests relaying finalized event certificates into a destination portal.
contract L1EventRegistry_RelayEvent_Test is L1EventRegistry_TestInit {
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
