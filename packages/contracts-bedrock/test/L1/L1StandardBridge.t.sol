// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { stdStorage, StdStorage } from "forge-std/StdStorage.sol";
import { stdError } from "forge-std/StdError.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";

// Contracts
import { ERC20 } from "@openzeppelin/contracts/token/ERC20/ERC20.sol";
import { StandardBridge } from "src/universal/StandardBridge.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { AddressAliasHelper } from "src/vendor/AddressAliasHelper.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";
import { IL1StandardBridge } from "interfaces/L1/IL1StandardBridge.sol";
import { ISystemConfig } from "interfaces/L1/ISystemConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";

/// @title L1StandardBridge_TestInit
/// @notice Reusable test initialization for `L1StandardBridge` tests.
abstract contract L1StandardBridge_TestInit is CommonTest {
    /// @notice Asserts the expected calls and events for bridging ETH depending on whether the
    ///         bridge call is legacy or not.
    function _preBridgeETH(bool isLegacy, uint256 value) internal {
        if (!isL1ForkTest()) {
            assertEq(address(optimismPortal2).balance, 0, "OptimismPortal2 balance should be 0");
        }
        uint256 nonce = l1CrossDomainMessenger.messageNonce();
        uint256 version = 0; // Internal constant in the OptimismPortal: DEPOSIT_VERSION
        address l1MessengerAliased = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        bytes memory message = abi.encodeCall(StandardBridge.finalizeBridgeETH, (alice, alice, value, hex"dead"));

        if (isLegacy) {
            vm.expectCall(
                address(l1StandardBridge), value, abi.encodeCall(l1StandardBridge.depositETH, (50000, hex"dead"))
            );
        } else {
            vm.expectCall(
                address(l1StandardBridge), value, abi.encodeCall(l1StandardBridge.bridgeETH, (50000, hex"dead"))
            );
        }
        vm.expectCall(
            address(l1CrossDomainMessenger),
            value,
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (address(l2StandardBridge), message, 50000))
        );

        bytes memory innerMessage = abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (nonce, address(l1StandardBridge), address(l2StandardBridge), value, 50000, message)
        );

        uint64 baseGas = l1CrossDomainMessenger.baseGas(message, 50000);
        vm.expectCall(
            address(optimismPortal2),
            value,
            abi.encodeCall(
                IOptimismPortal2.depositTransaction,
                (address(l2CrossDomainMessenger), value, baseGas, false, innerMessage)
            )
        );

        bytes memory opaqueData = abi.encodePacked(uint256(value), uint256(value), baseGas, false, innerMessage);

        vm.expectEmit(address(l1StandardBridge));
        emit ETHDepositInitiated(alice, alice, value, hex"dead");

        vm.expectEmit(address(l1StandardBridge));
        emit ETHBridgeInitiated(alice, alice, value, hex"dead");

        // OptimismPortal emits a TransactionDeposited event on `depositTransaction` call
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(l1MessengerAliased, address(l2CrossDomainMessenger), version, opaqueData);

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(address(l2StandardBridge), address(l1StandardBridge), message, nonce, 50000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessageExtension1(address(l1StandardBridge), value);

        vm.prank(alice, alice);
    }

    /// @notice Asserts the expected calls and events for bridging ETH to a different address
    ///         depending on whether the bridge call is legacy or not.
    function _preBridgeETHTo(bool isLegacy, uint256 value) internal {
        uint256 nonce = l1CrossDomainMessenger.messageNonce();
        uint256 version = 0; // Internal constant in the OptimismPortal: DEPOSIT_VERSION
        address l1MessengerAliased = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        if (isLegacy) {
            vm.expectCall(
                address(l1StandardBridge), value, abi.encodeCall(l1StandardBridge.depositETHTo, (bob, 60000, hex"dead"))
            );
        } else {
            vm.expectCall(
                address(l1StandardBridge), value, abi.encodeCall(l1StandardBridge.bridgeETHTo, (bob, 60000, hex"dead"))
            );
        }

        bytes memory message = abi.encodeCall(StandardBridge.finalizeBridgeETH, (alice, bob, value, hex"dead"));

        // the L1 bridge should call
        // L1CrossDomainMessenger.sendMessage
        vm.expectCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (address(l2StandardBridge), message, 60000))
        );

        bytes memory innerMessage = abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (nonce, address(l1StandardBridge), address(l2StandardBridge), value, 60000, message)
        );

        uint64 baseGas = l1CrossDomainMessenger.baseGas(message, 60000);
        vm.expectCall(
            address(optimismPortal2),
            abi.encodeCall(
                IOptimismPortal2.depositTransaction,
                (address(l2CrossDomainMessenger), value, baseGas, false, innerMessage)
            )
        );

        bytes memory opaqueData = abi.encodePacked(uint256(value), uint256(value), baseGas, false, innerMessage);

        vm.expectEmit(address(l1StandardBridge));
        emit ETHDepositInitiated(alice, bob, value, hex"dead");

        vm.expectEmit(address(l1StandardBridge));
        emit ETHBridgeInitiated(alice, bob, value, hex"dead");

        // OptimismPortal emits a TransactionDeposited event on `depositTransaction` call
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(l1MessengerAliased, address(l2CrossDomainMessenger), version, opaqueData);

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(address(l2StandardBridge), address(l1StandardBridge), message, nonce, 60000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessageExtension1(address(l1StandardBridge), value);

        // deposit eth to bob
        vm.prank(alice, alice);
    }
}

/// @title L1StandardBridge_Constructor_Test
/// @notice Tests the `constructor` function of the `L1StandardBridge` contract.
contract L1StandardBridge_Constructor_Test is CommonTest {
    /// @notice Test that the constructor sets the correct values.
    /// @dev Marked virtual to be overridden in test/kontrol/deployment/DeploymentSummary.t.sol
    function test_constructor_succeeds() external virtual {
        IL1StandardBridge impl = IL1StandardBridge(payable(EIP1967Helper.getImplementation(address(l1StandardBridge))));
        assertEq(address(impl.systemConfig()), address(0));

        // The constructor now uses _disableInitializers, whereas OP Mainnet has these values in
        // storage.
        returnIfForkTest("L1StandardBridge_Initialize_Test: impl storage differs on forked network");
        assertEq(address(impl.MESSENGER()), address(0));
        assertEq(address(impl.messenger()), address(0));
        assertEq(address(impl.OTHER_BRIDGE()), address(0));
        assertEq(address(impl.otherBridge()), address(0));
        assertEq(address(l2StandardBridge), Predeploys.L2_STANDARD_BRIDGE);
    }
}

/// @title L1StandardBridge_Initialize_Test
/// @notice Tests the `initialize` function of the `L1StandardBridge` contract.
contract L1StandardBridge_Initialize_Test is CommonTest {
    /// @notice Test that the initialize function sets the correct values.
    function test_initialize_succeeds() external view {
        assertEq(address(l1StandardBridge.systemConfig()), address(systemConfig));
        assertEq(address(l1StandardBridge.MESSENGER()), address(l1CrossDomainMessenger));
        assertEq(address(l1StandardBridge.messenger()), address(l1CrossDomainMessenger));
        assertEq(address(l1StandardBridge.OTHER_BRIDGE()), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(address(l1StandardBridge.otherBridge()), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(address(l2StandardBridge), Predeploys.L2_STANDARD_BRIDGE);
    }

    /// @notice Verifies initialize reverts with random unauthorized addresses
    /// @param _sender Random address for access control test
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        StorageSlot memory slot = ForgeArtifacts.getSlot("L1StandardBridge", "_initialized");
        vm.store(address(l1StandardBridge), bytes32(slot.slot), bytes32(0));

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);
        vm.prank(_sender);
        l1StandardBridge.initialize(l1CrossDomainMessenger, systemConfig);
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal initialization
    ///         but confirms that the initValue is not incremented incorrectly if an upgrade
    ///         function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("L1StandardBridge", "_initialized");

        // Get the initializer value.
        bytes32 slotVal = vm.load(address(l1StandardBridge), bytes32(slot.slot));
        uint8 val = uint8(uint256(slotVal) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, l1StandardBridge.initVersion());
    }
}

/// @title L1StandardBridge_Paused_Test
/// @notice Tests the `paused` function of the `L1StandardBridge` contract.
contract L1StandardBridge_Paused_Test is CommonTest {
    /// @notice Sets up the test by pausing the bridge, giving ether to the bridge and mocking the
    ///         calls to the xDomainMessageSender so that it returns the correct value.
    function _setupPausedBridge() internal {
        vm.startPrank(systemConfig.guardian());
        systemConfig.superchainConfig().pause(address(0));
        vm.stopPrank();
        assertTrue(l1StandardBridge.paused());

        vm.deal(address(l1StandardBridge.messenger()), 1 ether);

        vm.mockCall(
            address(l1StandardBridge.messenger()),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.otherBridge()))
        );
    }

    /// @notice Verifies that the `paused` accessor returns the same value as the `paused` function
    ///         of the `superchainConfig`.
    function test_paused_succeeds() external view {
        assertEq(l1StandardBridge.paused(), systemConfig.paused());
    }

    /// @notice Ensures that the `paused` function of the bridge contract actually calls the
    ///         `paused` function of the `superchainConfig`.
    function test_paused_callsSuperchainConfig_succeeds() external {
        vm.expectCall(address(systemConfig), abi.encodeCall(ISystemConfig.paused, ()));
        l1StandardBridge.paused();
    }

    /// @notice Checks that the `paused` state of the bridge matches the `paused` state of the
    ///         `superchainConfig` after it's been changed.
    function test_paused_matchesSuperchainConfig_succeeds() external {
        assertFalse(l1StandardBridge.paused());
        assertEq(l1StandardBridge.paused(), systemConfig.paused());

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        assertTrue(l1StandardBridge.paused());
        assertEq(l1StandardBridge.paused(), systemConfig.paused());
    }

    /// @notice Confirms that the `finalizeBridgeETH` function reverts when the bridge is paused.
    function test_paused_finalizeBridgeETH_reverts() external {
        _setupPausedBridge();

        vm.prank(address(l1StandardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1StandardBridge.finalizeBridgeETH{ value: 100 }({
            _from: address(2),
            _to: address(3),
            _amount: 100,
            _extraData: hex""
        });
    }

    /// @notice Confirms that the `finalizeETHWithdrawal` function reverts when the bridge is
    ///         paused.
    function test_paused_finalizeETHWithdrawal_reverts() external {
        _setupPausedBridge();

        vm.prank(address(l1StandardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1StandardBridge.finalizeETHWithdrawal{ value: 100 }({
            _from: address(2),
            _to: address(3),
            _amount: 100,
            _extraData: hex""
        });
    }

    /// @notice Confirms that the `finalizeERC20Withdrawal` function reverts when the bridge is
    ///         paused.
    function test_paused_finalizeERC20Withdrawal_reverts() external {
        _setupPausedBridge();

        vm.prank(address(l1StandardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1StandardBridge.finalizeERC20Withdrawal({
            _l1Token: address(0),
            _l2Token: address(0),
            _from: address(0),
            _to: address(0),
            _amount: 0,
            _extraData: hex""
        });
    }

    /// @notice Confirms that the `finalizeBridgeERC20` function reverts when the bridge is paused.
    function test_paused_finalizeBridgeERC20_reverts() external {
        _setupPausedBridge();

        vm.prank(address(l1StandardBridge.messenger()));
        vm.expectRevert("StandardBridge: paused");
        l1StandardBridge.finalizeBridgeERC20({
            _localToken: address(0),
            _remoteToken: address(0),
            _from: address(0),
            _to: address(0),
            _amount: 0,
            _extraData: hex""
        });
    }
}

/// @title L1StandardBridge_Receive_Test
/// @notice Tests the `receive` function of the `L1StandardBridge` contract.
contract L1StandardBridge_Receive_Test is CommonTest {
    /// @notice Tests receive bridges ETH successfully.
    function test_receive_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;

        // The legacy event must be emitted for backwards compatibility
        vm.expectEmit(address(l1StandardBridge));
        emit ETHDepositInitiated(alice, alice, 100, hex"");

        vm.expectEmit(address(l1StandardBridge));
        emit ETHBridgeInitiated(alice, alice, 100, hex"");

        vm.expectCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage,
                (
                    address(l2StandardBridge),
                    abi.encodeCall(StandardBridge.finalizeBridgeETH, (alice, alice, 100, hex"")),
                    200_000
                )
            )
        );

        vm.prank(alice, alice);
        (bool success,) = address(l1StandardBridge).call{ value: 100 }(hex"");
        assertEq(success, true);

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 100);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 100);
        }
    }

    /// @notice Verifies receive function reverts when called by contracts
    function test_receive_notEOA_reverts() external {
        vm.etch(alice, hex"ffff");
        vm.deal(alice, 100);
        vm.prank(alice);
        vm.expectRevert(bytes("StandardBridge: function can only be called from an EOA"));
        (bool revertsAsExpected,) = address(l1StandardBridge).call{ value: 100 }(hex"");
        assertTrue(revertsAsExpected, "expectRevert: call did not revert");
    }

    /// @notice Tests that receive reverts when custom gas token is enabled and value is sent.
    function testFuzz_receive_withCustomGasToken_reverts(uint256 _value) external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);

        vm.prank(alice, alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);

        (bool revertsAsExpected,) = address(l1StandardBridge).call{ value: _value }(hex"");
        assertTrue(revertsAsExpected, "expectRevert: call did not revert");
    }
}

/// @title L1StandardBridge_DepositETH_Test
/// @notice Tests the `depositETH` function of the `L1StandardBridge` contract.
contract L1StandardBridge_DepositETH_Test is L1StandardBridge_TestInit {
    /// @notice Tests that depositing ETH succeeds.
    ///         Emits ETHDepositInitiated and ETHBridgeInitiated events.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         Only EOA can call depositETH.
    ///         ETH ends up in the optimismPortal.
    function test_depositETH_fromEOA_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        _preBridgeETH({ isLegacy: true, value: 500 });
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;
        l1StandardBridge.depositETH{ value: 500 }(50000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 500);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 500);
        }
    }

    /// @notice Tests that depositing ETH succeeds for an EOA using 7702 delegation.
    function test_depositETH_fromEOA7702_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        // Set alice to have 7702 code.
        vm.etch(alice, abi.encodePacked(hex"EF0100", address(0)));

        _preBridgeETH({ isLegacy: true, value: 500 });
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;
        l1StandardBridge.depositETH{ value: 500 }(50000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 500);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 500);
        }
    }

    /// @notice Tests that depositing ETH reverts if the call is not from an EOA.
    function test_depositETH_notEoa_reverts() external {
        vm.etch(alice, address(L1Token).code);
        vm.expectRevert("StandardBridge: function can only be called from an EOA");
        vm.prank(alice);
        l1StandardBridge.depositETH{ value: 1 }(300, hex"");
    }

    /// @notice Tests that depositETH reverts when custom gas token is enabled and value is sent.
    function testFuzz_depositETH_withCustomGasToken_reverts(uint256 _value, uint32 _minGasLimit) external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);

        vm.prank(alice, alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);
        l1StandardBridge.depositETH{ value: _value }(_minGasLimit, hex"dead");
    }

    /// @notice Tests that depositETH reverts when custom gas token is enabled for EOA with 7702 delegation.
    function testFuzz_depositETH_fromEOA7702WithCustomGasToken_reverts(uint256 _value, uint32 _minGasLimit) external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
        _value = bound(_value, 1, type(uint128).max);

        // Set alice to have 7702 code.
        vm.etch(alice, abi.encodePacked(hex"EF0100", address(0)));

        vm.deal(alice, _value);
        vm.prank(alice, alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);
        l1StandardBridge.depositETH{ value: _value }(_minGasLimit, hex"dead");
    }
}

/// @title L1StandardBridge_DepositETHTo_Test
/// @notice Tests the `depositETHTo` function of the `L1StandardBridge` contract.
contract L1StandardBridge_DepositETHTo_Test is L1StandardBridge_TestInit {
    /// @notice Tests that depositing ETH to a different address succeeds.
    ///         Emits ETHDepositInitiated event.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         EOA or contract can call depositETHTo.
    ///         ETH ends up in the optimismPortal.
    function test_depositETHTo_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        _preBridgeETHTo({ isLegacy: true, value: 600 });
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;
        l1StandardBridge.depositETHTo{ value: 600 }(bob, 60000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 600);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 600);
        }
    }

    /// @notice Verifies depositETHTo succeeds with various recipients and amounts
    /// @param _to Random recipient address
    /// @param _amount Random ETH amount to deposit
    function testFuzz_depositETHTo_randomRecipient_succeeds(address _to, uint256 _amount) external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        vm.assume(_to != address(0));
        _amount = bound(_amount, 1, 10 ether);

        vm.deal(alice, _amount);

        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;

        vm.prank(alice);
        l1StandardBridge.depositETHTo{ value: _amount }(_to, 60000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + _amount);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + _amount);
        }
    }

    /// @notice Tests that depositETHTo reverts when custom gas token is enabled and value is sent.
    function testFuzz_depositETHTo_withCustomGasToken_reverts(
        address _to,
        uint256 _value,
        uint32 _minGasLimit
    )
        external
    {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);
        vm.assume(_to != address(0));
        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);
        vm.prank(alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);
        l1StandardBridge.depositETHTo{ value: _value }(_to, _minGasLimit, hex"dead");
    }
}

/// @title L1StandardBridge_DepositERC20_Test
/// @notice Tests the `depositERC20` function of the `L1StandardBridge` contract.
contract L1StandardBridge_DepositERC20_Test is CommonTest {
    using stdStorage for StdStorage;

    // depositERC20
    // - updates bridge.deposits
    // - emits ERC20DepositInitiated
    // - calls optimismPortal.depositTransaction
    // - only callable by EOA

    /// @notice Tests that depositing ERC20 to the bridge succeeds.
    ///         Bridge deposits are updated.
    ///         Emits ERC20DepositInitiated event.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         Only EOA can call depositERC20.
    function test_depositERC20_succeeds() external {
        uint256 nonce = l1CrossDomainMessenger.messageNonce();
        uint256 version = 0; // Internal constant in the OptimismPortal: DEPOSIT_VERSION
        address l1MessengerAliased = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        // Deal Alice's ERC20 State
        deal(address(L1Token), alice, 100000, true);
        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), type(uint256).max);

        // The l1StandardBridge should transfer alice's tokens to itself
        vm.expectCall(address(L1Token), abi.encodeCall(ERC20.transferFrom, (alice, address(l1StandardBridge), 100)));

        bytes memory message = abi.encodeCall(
            StandardBridge.finalizeBridgeERC20, (address(L2Token), address(L1Token), alice, alice, 100, hex"")
        );

        // the L1 bridge should call L1CrossDomainMessenger.sendMessage
        vm.expectCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (address(l2StandardBridge), message, 10000))
        );

        bytes memory innerMessage = abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (nonce, address(l1StandardBridge), address(l2StandardBridge), 0, 10000, message)
        );

        uint64 baseGas = l1CrossDomainMessenger.baseGas(message, 10000);
        vm.expectCall(
            address(optimismPortal2),
            abi.encodeCall(
                IOptimismPortal2.depositTransaction, (address(l2CrossDomainMessenger), 0, baseGas, false, innerMessage)
            )
        );

        bytes memory opaqueData = abi.encodePacked(uint256(0), uint256(0), baseGas, false, innerMessage);

        // Should emit both the bedrock and legacy events
        vm.expectEmit(address(l1StandardBridge));
        emit ERC20DepositInitiated(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.expectEmit(address(l1StandardBridge));
        emit ERC20BridgeInitiated(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        // OptimismPortal emits a TransactionDeposited event on `depositTransaction` call
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(l1MessengerAliased, address(l2CrossDomainMessenger), version, opaqueData);

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(address(l2StandardBridge), address(l1StandardBridge), message, nonce, 10000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessageExtension1(address(l1StandardBridge), 0);

        vm.prank(alice, alice);
        l1StandardBridge.depositERC20(address(L1Token), address(L2Token), 100, 10000, hex"");
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 100);
    }

    /// @notice Tests that depositing an ERC20 to the bridge reverts if the caller is not an EOA.
    function test_depositERC20_notEoa_reverts() external {
        // Turn alice into a contract
        vm.etch(alice, hex"ffff");

        vm.expectRevert("StandardBridge: function can only be called from an EOA");
        vm.prank(alice);
        l1StandardBridge.depositERC20(address(0), address(0), 100, 100, hex"");
    }

    /// @notice Verifies depositERC20 succeeds with various amounts and gas limits
    /// @param _amount Random ERC20 amount to deposit
    /// @param _gasLimit Random gas limit for L2 execution
    function testFuzz_depositERC20_amountAndGas_succeeds(uint256 _amount, uint32 _gasLimit) external {
        _amount = bound(_amount, 1, 1000000);
        _gasLimit = uint32(bound(uint256(_gasLimit), 21000, 10000000));

        deal(address(L1Token), alice, _amount, true);
        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), _amount);

        vm.prank(alice, alice);
        l1StandardBridge.depositERC20(address(L1Token), address(L2Token), _amount, _gasLimit, hex"");
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), _amount);
    }
}

/// @title L1StandardBridge_DepositERC20To_Test
/// @notice Tests the `depositERC20To` function of the `L1StandardBridge` contract.
contract L1StandardBridge_DepositERC20To_Test is CommonTest {
    /// @notice Tests that depositing ERC20 to the bridge succeeds when sent to a different address.
    ///         Bridge deposits are updated.
    ///         Emits ERC20DepositInitiated event.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         Contracts can call depositERC20.
    function test_depositERC20To_succeeds() external {
        uint256 nonce = l1CrossDomainMessenger.messageNonce();
        uint256 version = 0; // Internal constant in the OptimismPortal: DEPOSIT_VERSION
        address l1MessengerAliased = AddressAliasHelper.applyL1ToL2Alias(address(l1CrossDomainMessenger));

        bytes memory message = abi.encodeCall(
            StandardBridge.finalizeBridgeERC20, (address(L2Token), address(L1Token), alice, bob, 1000, hex"")
        );

        bytes memory innerMessage = abi.encodeCall(
            ICrossDomainMessenger.relayMessage,
            (nonce, address(l1StandardBridge), address(l2StandardBridge), 0, 10000, message)
        );

        uint64 baseGas = l1CrossDomainMessenger.baseGas(message, 10000);
        bytes memory opaqueData = abi.encodePacked(uint256(0), uint256(0), baseGas, false, innerMessage);

        deal(address(L1Token), alice, 100000, true);

        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), type(uint256).max);

        // Should emit both the bedrock and legacy events
        vm.expectEmit(address(l1StandardBridge));
        emit ERC20DepositInitiated(address(L1Token), address(L2Token), alice, bob, 1000, hex"");

        vm.expectEmit(address(l1StandardBridge));
        emit ERC20BridgeInitiated(address(L1Token), address(L2Token), alice, bob, 1000, hex"");

        // OptimismPortal emits a TransactionDeposited event on `depositTransaction` call
        vm.expectEmit(address(optimismPortal2));
        emit TransactionDeposited(l1MessengerAliased, address(l2CrossDomainMessenger), version, opaqueData);

        // SentMessage event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessage(address(l2StandardBridge), address(l1StandardBridge), message, nonce, 10000);

        // SentMessageExtension1 event emitted by the CrossDomainMessenger
        vm.expectEmit(address(l1CrossDomainMessenger));
        emit SentMessageExtension1(address(l1StandardBridge), 0);

        // the L1 bridge should call L1CrossDomainMessenger.sendMessage
        vm.expectCall(
            address(l1CrossDomainMessenger),
            abi.encodeCall(ICrossDomainMessenger.sendMessage, (address(l2StandardBridge), message, 10000))
        );
        // The L1 XDM should call OptimismPortal.depositTransaction
        vm.expectCall(
            address(optimismPortal2),
            abi.encodeCall(
                IOptimismPortal2.depositTransaction, (address(l2CrossDomainMessenger), 0, baseGas, false, innerMessage)
            )
        );
        vm.expectCall(address(L1Token), abi.encodeCall(ERC20.transferFrom, (alice, address(l1StandardBridge), 1000)));

        vm.prank(alice);
        l1StandardBridge.depositERC20To(address(L1Token), address(L2Token), bob, 1000, 10000, hex"");

        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 1000);
    }

    /// @notice Verifies depositERC20To succeeds with zero amount
    function test_depositERC20To_zeroAmount_succeeds() external {
        deal(address(L1Token), alice, 1000, true);
        vm.prank(alice);
        L1Token.approve(address(l1StandardBridge), 0);

        vm.prank(alice);
        l1StandardBridge.depositERC20To(address(L1Token), address(L2Token), bob, 0, 10000, hex"");
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 0);
    }
}

/// @title L1StandardBridge_FinalizeETHWithdrawal_Test
/// @notice Tests the `finalizeETHWithdrawal` function of the `L1StandardBridge` contract.
contract L1StandardBridge_FinalizeETHWithdrawal_Test is CommonTest {
    using stdStorage for StdStorage;

    /// @notice Tests that finalizing an ETH withdrawal succeeds.
    ///         Emits ETHWithdrawalFinalized event.
    ///         Only callable by the L2 bridge.
    function test_finalizeETHWithdrawal_succeeds() external {
        uint256 aliceBalance = alice.balance;

        vm.expectEmit(address(l1StandardBridge));
        emit ETHWithdrawalFinalized(alice, alice, 100, hex"");

        vm.expectEmit(address(l1StandardBridge));
        emit ETHBridgeFinalized(alice, alice, 100, hex"");

        vm.expectCall(alice, hex"");

        vm.mockCall(
            address(l1StandardBridge.messenger()),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        // ensure that the messenger has ETH to call with
        vm.deal(address(l1StandardBridge.messenger()), 100);
        vm.prank(address(l1StandardBridge.messenger()));
        l1StandardBridge.finalizeETHWithdrawal{ value: 100 }(alice, alice, 100, hex"");

        assertEq(address(l1StandardBridge.messenger()).balance, 0);
        assertEq(aliceBalance + 100, alice.balance);
    }
}

/// @title L1StandardBridge_FinalizeERC20Withdrawal_Test
/// @notice Tests the `finalizeERC20Withdrawal` function of the `L1StandardBridge` contract.
contract L1StandardBridge_FinalizeERC20Withdrawal_Test is CommonTest {
    using stdStorage for StdStorage;

    /// @notice Tests that finalizing an ERC20 withdrawal succeeds.
    ///         Bridge deposits are updated.
    ///         Emits ERC20WithdrawalFinalized event.
    ///         Only callable by the L2 bridge.
    function test_finalizeERC20Withdrawal_succeeds() external {
        deal(address(L1Token), address(l1StandardBridge), 100, true);

        uint256 slot = stdstore.target(address(l1StandardBridge)).sig("deposits(address,address)").with_key(
            address(L1Token)
        ).with_key(address(L2Token)).find();

        // Give the L1 bridge some ERC20 tokens
        vm.store(address(l1StandardBridge), bytes32(slot), bytes32(uint256(100)));
        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), 100);

        vm.expectEmit(address(l1StandardBridge));
        emit ERC20WithdrawalFinalized(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.expectEmit(address(l1StandardBridge));
        emit ERC20BridgeFinalized(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        vm.expectCall(address(L1Token), abi.encodeCall(ERC20.transfer, (alice, 100)));

        vm.mockCall(
            address(l1StandardBridge.messenger()),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.prank(address(l1StandardBridge.messenger()));
        l1StandardBridge.finalizeERC20Withdrawal(address(L1Token), address(L2Token), alice, alice, 100, hex"");

        assertEq(L1Token.balanceOf(address(l1StandardBridge)), 0);
        assertEq(L1Token.balanceOf(address(alice)), 100);
    }

    /// @notice Verifies finalizeERC20Withdrawal reverts with unauthorized messenger
    /// @param _caller Random address that is not the messenger
    function testFuzz_finalizeERC20Withdrawal_notMessenger_reverts(address _caller) external {
        vm.assume(_caller != address(l1StandardBridge.messenger()));

        vm.mockCall(
            address(l1StandardBridge.messenger()),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.expectRevert("StandardBridge: function can only be called from the other bridge");
        vm.prank(_caller);
        l1StandardBridge.finalizeERC20Withdrawal(address(L1Token), address(L2Token), alice, alice, 100, hex"");
    }

    /// @notice Tests that finalizing an ERC20 withdrawal reverts if the caller is not the L2
    ///         bridge.
    function test_finalizeERC20Withdrawal_notOtherBridge_reverts() external {
        vm.mockCall(
            address(l1StandardBridge.messenger()),
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(0))
        );
        vm.prank(address(l1StandardBridge.messenger()));
        vm.expectRevert("StandardBridge: function can only be called from the other bridge");
        l1StandardBridge.finalizeERC20Withdrawal(address(L1Token), address(L2Token), alice, alice, 100, hex"");
    }
}

/// @title L1StandardBridge_Uncategorized_Test
/// @notice General tests that are not testing any function directly of the `L1StandardBridge`
///         contract or are testing multiple functions.
contract L1StandardBridge_Uncategorized_Test is L1StandardBridge_TestInit {
    /// @notice Test that the accessors return the correct initialized values.
    function test_getters_succeeds() external view {
        assert(l1StandardBridge.l2TokenBridge() == address(l2StandardBridge));
        assert(address(l1StandardBridge.OTHER_BRIDGE()) == address(l2StandardBridge));
        assert(address(l1StandardBridge.messenger()) == address(l1CrossDomainMessenger));
        assert(address(l1StandardBridge.MESSENGER()) == address(l1CrossDomainMessenger));
        assert(l1StandardBridge.systemConfig() == systemConfig);
        assert(l1StandardBridge.superchainConfig() == systemConfig.superchainConfig());
    }

    /// @notice Tests that bridging ETH succeeds.
    ///         Emits ETHDepositInitiated and ETHBridgeInitiated events.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         Only EOA can call bridgeETH.
    ///         ETH ends up in the optimismPortal.
    function test_bridgeETH_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        _preBridgeETH({ isLegacy: false, value: 500 });
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;
        l1StandardBridge.bridgeETH{ value: 500 }(50000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 500);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 500);
        }
    }

    /// @notice Tests that bridging ETH to a different address succeeds.
    ///         Emits ETHDepositInitiated and ETHBridgeInitiated events.
    ///         Calls depositTransaction on the OptimismPortal.
    ///         Only EOA can call bridgeETHTo.
    ///         ETH ends up in the optimismPortal.
    function test_bridgeETHTo_succeeds() external {
        skipIfSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN);
        _preBridgeETHTo({ isLegacy: false, value: 600 });
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;
        l1StandardBridge.bridgeETHTo{ value: 600 }(bob, 60000, hex"dead");

        if (isSysFeatureEnabled(Features.ETH_LOCKBOX)) {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore);
            assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + 600);
        } else {
            assertEq(address(optimismPortal2).balance, portalBalanceBefore + 600);
        }
    }

    /// @notice Tests that finalizing bridged ETH succeeds.
    function test_finalizeBridgeETH_succeeds() external {
        address messenger = address(l1StandardBridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);

        vm.expectEmit(address(l1StandardBridge));
        emit ETHBridgeFinalized(alice, alice, 100, hex"");

        l1StandardBridge.finalizeBridgeETH{ value: 100 }(alice, alice, 100, hex"");
    }

    /// @notice Tests that finalizing bridged ETH reverts if the amount is incorrect.
    function test_finalizeBridgeETH_incorrectValue_reverts() external {
        address messenger = address(l1StandardBridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: amount sent does not match amount required");
        l1StandardBridge.finalizeBridgeETH{ value: 50 }(alice, alice, 100, hex"");
    }

    /// @notice Tests that finalizing bridged ETH reverts if the destination is the L1 bridge.
    function test_finalizeBridgeETH_sendToSelf_reverts() external {
        address messenger = address(l1StandardBridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: cannot send to self");
        l1StandardBridge.finalizeBridgeETH{ value: 100 }(alice, address(l1StandardBridge), 100, hex"");
    }

    /// @notice Tests that finalizing bridged ETH reverts if the destination is the messenger.
    function test_finalizeBridgeETH_sendToMessenger_reverts() external {
        address messenger = address(l1StandardBridge.messenger());
        vm.mockCall(
            messenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
        vm.deal(messenger, 100);
        vm.prank(messenger);
        vm.expectRevert("StandardBridge: cannot send to messenger");
        l1StandardBridge.finalizeBridgeETH{ value: 100 }(alice, messenger, 100, hex"");
    }

    /// @notice Tests that bridgeETH reverts when custom gas token is enabled and value is sent.
    function testFuzz_bridgeETH_withCustomGasToken_reverts(uint256 _value, uint32 _minGasLimit) external {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);

        vm.prank(alice, alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);
        l1StandardBridge.bridgeETH{ value: _value }(_minGasLimit, hex"dead");
    }

    /// @notice Tests that bridgeETHTo reverts when custom gas token is enabled and value is sent.
    function testFuzz_bridgeETHTo_withCustomGasToken_reverts(
        address _to,
        uint256 _value,
        uint32 _minGasLimit
    )
        external
    {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        vm.assume(_to != address(0));
        _value = bound(_value, 1, type(uint128).max);
        vm.deal(alice, _value);

        vm.prank(alice);
        vm.expectRevert(IOptimismPortal2.OptimismPortal_NotAllowedOnCGTMode.selector);
        l1StandardBridge.bridgeETHTo{ value: _value }(_to, _minGasLimit, hex"dead");
    }
}

/// @title L1StandardBridge_WithdrawalThrottle_TestInit
/// @notice Reusable test initialization for L1StandardBridge withdrawal throttle tests.
abstract contract L1StandardBridge_WithdrawalThrottle_TestInit is CommonTest {
    using stdStorage for StdStorage;

    uint16 internal constant MAX_BPS = 1000;
    uint64 internal constant REFILL_PERIOD = 100;
    uint256 internal constant STOCK = 1000;
    uint256 internal constant CAPACITY = 100;

    address internal bridgeMessenger;
    address internal bridgeOwner;

    /// @notice Sets up an escrowed token and an authenticated L2 bridge caller.
    function setUp() public virtual override {
        super.setUp();

        bridgeMessenger = address(l1StandardBridge.messenger());
        bridgeOwner = l1StandardBridge.proxyAdminOwner();
        deal(address(L1Token), address(l1StandardBridge), STOCK, true);
        _setDeposits(address(L1Token), address(L2Token), STOCK);
        vm.mockCall(
            bridgeMessenger,
            abi.encodeCall(ICrossDomainMessenger.xDomainMessageSender, ()),
            abi.encode(address(l1StandardBridge.OTHER_BRIDGE()))
        );
    }

    /// @notice Configures the default L1 token's withdrawal throttle.
    function _configureThrottle() internal {
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), MAX_BPS, REFILL_PERIOD);
    }

    /// @notice Sets the escrow accounting for a local and remote token pair.
    function _setDeposits(address _localToken, address _remoteToken, uint256 _amount) internal {
        uint256 slot = stdstore.target(address(l1StandardBridge)).sig("deposits(address,address)").with_key(_localToken)
            .with_key(_remoteToken).find();
        vm.store(address(l1StandardBridge), bytes32(slot), bytes32(_amount));
    }

    /// @notice Finalizes an ERC20 withdrawal through the authenticated messenger.
    function _withdraw(address _localToken, address _remoteToken, uint256 _amount) internal {
        vm.prank(bridgeMessenger);
        l1StandardBridge.finalizeBridgeERC20(_localToken, _remoteToken, alice, alice, _amount, hex"");
    }
}

/// @title L1StandardBridge_SetWithdrawalThrottle_Test
/// @notice Tests the `setWithdrawalThrottle` function of the `L1StandardBridge` contract.
contract L1StandardBridge_SetWithdrawalThrottle_Test is L1StandardBridge_WithdrawalThrottle_TestInit {
    /// @notice Tests that configuring a withdrawal throttle snapshots stock and starts full.
    function test_setWithdrawalThrottle_succeeds() external {
        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleConfigured(address(L1Token), MAX_BPS, REFILL_PERIOD, STOCK, CAPACITY, CAPACITY);

        _configureThrottle();

        IL1StandardBridge.WithdrawalThrottleConfig memory throttle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertEq(throttle.capacity, CAPACITY);
        assertEq(throttle.available, CAPACITY);
        assertEq(throttle.refillPeriod, REFILL_PERIOD);
        assertEq(throttle.lastUpdated, block.timestamp);
        assertEq(throttle.maxBps, MAX_BPS);
        assertTrue(throttle.enabled);
        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), CAPACITY);
    }

    /// @notice Tests that only the ProxyAdmin owner can configure a withdrawal throttle.
    function testFuzz_setWithdrawalThrottle_notProxyAdminOwner_reverts(address _caller) external {
        vm.assume(_caller != bridgeOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), MAX_BPS, REFILL_PERIOD);
    }

    /// @notice Tests that configuring the zero token address reverts.
    function test_setWithdrawalThrottle_zeroToken_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_InvalidWithdrawalThrottleToken.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(0), MAX_BPS, REFILL_PERIOD);
    }

    /// @notice Tests that configuring a mintable local token reverts because it is not escrowed.
    function test_setWithdrawalThrottle_mintableToken_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_UnsupportedWithdrawalThrottleToken.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(BadL1Token), MAX_BPS, REFILL_PERIOD);
    }

    /// @notice Tests that a zero basis point value reverts.
    function test_setWithdrawalThrottle_zeroBps_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_InvalidWithdrawalThrottleBps.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), 0, REFILL_PERIOD);
    }

    /// @notice Tests that a basis point value above 100% reverts.
    function test_setWithdrawalThrottle_bpsAboveMaximum_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_InvalidWithdrawalThrottleBps.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), 10_001, REFILL_PERIOD);
    }

    /// @notice Tests that a zero refill period reverts.
    function test_setWithdrawalThrottle_zeroRefillPeriod_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_InvalidWithdrawalThrottlePeriod.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), MAX_BPS, 0);
    }

    /// @notice Tests that reconfiguration preserves and clamps accrued capacity under new parameters.
    function test_setWithdrawalThrottle_reconfigurePreservesAvailable_succeeds() external {
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), 60);
        vm.warp(block.timestamp + REFILL_PERIOD / 2);

        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), 500, REFILL_PERIOD * 2);

        IL1StandardBridge.WithdrawalThrottleConfig memory throttle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertEq(throttle.capacity, 47);
        assertEq(throttle.available, 47);
        assertEq(throttle.refillPeriod, REFILL_PERIOD * 2);
        assertEq(throttle.lastUpdated, block.timestamp);
        assertEq(throttle.refillRemainder, 0);
        assertEq(throttle.maxBps, 500);
    }
}

/// @title L1StandardBridge_RefreshWithdrawalThrottle_Test
/// @notice Tests the `refreshWithdrawalThrottle` function of the `L1StandardBridge` contract.
contract L1StandardBridge_RefreshWithdrawalThrottle_Test is L1StandardBridge_WithdrawalThrottle_TestInit {
    /// @notice Tests that refreshing increased stock does not refill the bucket.
    function test_refreshWithdrawalThrottle_increasedStockPreservesAvailable_succeeds() external {
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), 60);
        deal(address(L1Token), address(l1StandardBridge), 2000, true);

        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleRefreshed(address(L1Token), 2000, 200, 40);
        vm.prank(bridgeOwner);
        l1StandardBridge.refreshWithdrawalThrottle(address(L1Token));

        IL1StandardBridge.WithdrawalThrottleConfig memory throttle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertEq(throttle.capacity, 200);
        assertEq(throttle.available, 40);
    }

    /// @notice Tests that refreshing decreased stock clamps available capacity.
    function test_refreshWithdrawalThrottle_decreasedStockClampsAvailable_succeeds() external {
        _configureThrottle();
        deal(address(L1Token), address(l1StandardBridge), 50, true);

        vm.prank(bridgeOwner);
        l1StandardBridge.refreshWithdrawalThrottle(address(L1Token));

        IL1StandardBridge.WithdrawalThrottleConfig memory throttle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertEq(throttle.capacity, 5);
        assertEq(throttle.available, 5);
    }

    /// @notice Tests that refreshing a disabled throttle reverts.
    function test_refreshWithdrawalThrottle_notEnabled_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_WithdrawalThrottleNotEnabled.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.refreshWithdrawalThrottle(address(L1Token));
    }

    /// @notice Tests that only the ProxyAdmin owner can refresh a withdrawal throttle.
    function testFuzz_refreshWithdrawalThrottle_notProxyAdminOwner_reverts(address _caller) external {
        _configureThrottle();
        vm.assume(_caller != bridgeOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        l1StandardBridge.refreshWithdrawalThrottle(address(L1Token));
    }
}

/// @title L1StandardBridge_DisableWithdrawalThrottle_Test
/// @notice Tests the `disableWithdrawalThrottle` function of the `L1StandardBridge` contract.
contract L1StandardBridge_DisableWithdrawalThrottle_Test is L1StandardBridge_WithdrawalThrottle_TestInit {
    /// @notice Tests that disabling removes the throttle and restores unrestricted withdrawals.
    function test_disableWithdrawalThrottle_succeeds() external {
        _configureThrottle();

        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleDisabled(address(L1Token));
        vm.prank(bridgeOwner);
        l1StandardBridge.disableWithdrawalThrottle(address(L1Token));

        IL1StandardBridge.WithdrawalThrottleConfig memory throttle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertFalse(throttle.enabled);
        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), type(uint256).max);

        _withdraw(address(L1Token), address(L2Token), CAPACITY + 1);
        assertEq(L1Token.balanceOf(alice), CAPACITY + 1);
    }

    /// @notice Tests that disabling an absent throttle reverts.
    function test_disableWithdrawalThrottle_notEnabled_reverts() external {
        vm.expectRevert(IL1StandardBridge.L1StandardBridge_WithdrawalThrottleNotEnabled.selector);
        vm.prank(bridgeOwner);
        l1StandardBridge.disableWithdrawalThrottle(address(L1Token));
    }

    /// @notice Tests that only the ProxyAdmin owner can disable a withdrawal throttle.
    function testFuzz_disableWithdrawalThrottle_notProxyAdminOwner_reverts(address _caller) external {
        _configureThrottle();
        vm.assume(_caller != bridgeOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        l1StandardBridge.disableWithdrawalThrottle(address(L1Token));
    }
}

/// @title L1StandardBridge_FinalizeBridgeERC20_Test
/// @notice Tests withdrawal throttle behavior during ERC20 finalization.
contract L1StandardBridge_FinalizeBridgeERC20_Test is L1StandardBridge_WithdrawalThrottle_TestInit {
    /// @notice Tests that an unconfigured token remains unrestricted.
    function test_finalizeBridgeERC20_throttleDisabled_succeeds() external {
        _withdraw(address(L1Token), address(L2Token), CAPACITY + 1);

        assertEq(L1Token.balanceOf(alice), CAPACITY + 1);
        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), type(uint256).max);
    }

    /// @notice Tests that consuming exactly the bucket capacity succeeds and emits monitoring events.
    function test_finalizeBridgeERC20_exactCapacity_succeeds() external {
        _configureThrottle();

        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleCapacityConsumed(address(L1Token), CAPACITY, 0);
        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleCapacityExhausted(address(L1Token));
        _withdraw(address(L1Token), address(L2Token), CAPACITY);

        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), 0);
        assertEq(L1Token.balanceOf(alice), CAPACITY);
    }

    /// @notice Tests that a withdrawal reverts when its bucket does not have enough capacity.
    function test_finalizeBridgeERC20_insufficientCapacity_reverts() external {
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), CAPACITY);

        vm.expectRevert(
            abi.encodeWithSelector(
                IL1StandardBridge.L1StandardBridge_WithdrawalThrottled.selector, address(L1Token), 1, 0, CAPACITY
            )
        );
        _withdraw(address(L1Token), address(L2Token), 1);

        assertEq(l1StandardBridge.deposits(address(L1Token), address(L2Token)), STOCK - CAPACITY);
        assertEq(L1Token.balanceOf(alice), CAPACITY);
    }

    /// @notice Tests that withdrawal capacity refills linearly over the configured period.
    function test_finalizeBridgeERC20_refillsOverTime_succeeds() external {
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), CAPACITY);

        uint256 emptiedAt = block.timestamp;
        vm.warp(emptiedAt + REFILL_PERIOD / 2);
        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), CAPACITY / 2);
        _withdraw(address(L1Token), address(L2Token), CAPACITY / 2);

        vm.warp(block.timestamp + REFILL_PERIOD);
        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), CAPACITY);
    }

    /// @notice Tests that staggered withdrawals preserve fractional refill capacity.
    function test_finalizeBridgeERC20_fractionalRefillRemainder_succeeds() external {
        vm.prank(bridgeOwner);
        l1StandardBridge.setWithdrawalThrottle(address(L1Token), MAX_BPS, 3);
        _withdraw(address(L1Token), address(L2Token), CAPACITY);

        vm.warp(block.timestamp + 1);
        _withdraw(address(L1Token), address(L2Token), 33);
        vm.warp(block.timestamp + 1);

        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), 33);
        _withdraw(address(L1Token), address(L2Token), 33);
        vm.warp(block.timestamp + 1);

        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), 34);
        _withdraw(address(L1Token), address(L2Token), 34);
    }

    /// @notice Tests the monitoring event emitted by a partial capacity consumption.
    function test_finalizeBridgeERC20_partialCapacityEmitsRemaining_succeeds() external {
        _configureThrottle();

        vm.expectEmit(address(l1StandardBridge));
        emit WithdrawalThrottleCapacityConsumed(address(L1Token), 40, 60);
        _withdraw(address(L1Token), address(L2Token), 40);

        assertEq(l1StandardBridge.availableWithdrawalCapacity(address(L1Token)), 60);
    }

    /// @notice Tests that all remote representations of an L1 token share one bucket.
    function test_finalizeBridgeERC20_remoteRepresentationsShareBucket_reverts() external {
        address alternateRemoteToken = makeAddr("alternateRemoteToken");
        _setDeposits(address(L1Token), alternateRemoteToken, STOCK);
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), 60);

        vm.expectRevert(
            abi.encodeWithSelector(
                IL1StandardBridge.L1StandardBridge_WithdrawalThrottled.selector, address(L1Token), 41, 40, CAPACITY
            )
        );
        _withdraw(address(L1Token), alternateRemoteToken, 41);
    }

    /// @notice Tests that a configured token's bucket does not affect another L1 token.
    function test_finalizeBridgeERC20_tokensHaveIndependentBuckets_succeeds() external {
        ERC20 otherToken = new ERC20("Other Token", "OTHR");
        address remoteOtherToken = makeAddr("remoteOtherToken");
        deal(address(otherToken), address(l1StandardBridge), STOCK, true);
        _setDeposits(address(otherToken), remoteOtherToken, STOCK);
        _configureThrottle();
        _withdraw(address(L1Token), address(L2Token), CAPACITY);

        _withdraw(address(otherToken), remoteOtherToken, CAPACITY + 1);

        assertEq(otherToken.balanceOf(alice), CAPACITY + 1);
    }

    /// @notice Tests that capacity consumption rolls back if escrow accounting later fails.
    function test_finalizeBridgeERC20_laterFailureRollsBackConsumption_reverts() external {
        _configureThrottle();
        IL1StandardBridge.WithdrawalThrottleConfig memory beforeThrottle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        _setDeposits(address(L1Token), address(L2Token), 10);

        vm.expectRevert(stdError.arithmeticError);
        _withdraw(address(L1Token), address(L2Token), CAPACITY / 2);

        IL1StandardBridge.WithdrawalThrottleConfig memory afterThrottle =
            l1StandardBridge.withdrawalThrottle(address(L1Token));
        assertEq(afterThrottle.available, beforeThrottle.available);
        assertEq(afterThrottle.lastUpdated, beforeThrottle.lastUpdated);
    }
}
