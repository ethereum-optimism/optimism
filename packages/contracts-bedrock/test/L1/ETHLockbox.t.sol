// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Contracts
import { Proxy } from "src/universal/Proxy.sol";

// Libraries
import { Constants } from "src/libraries/Constants.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IETHLockbox } from "interfaces/L1/IETHLockbox.sol";
import { IProxyAdminOwnedBase } from "interfaces/universal/IProxyAdminOwnedBase.sol";
import { IOptimismPortal2 } from "interfaces/L1/IOptimismPortal2.sol";

/// @title ETHLockbox_TestInit
/// @notice Base contract that sets up the testing environment for ETHLockbox tests.
abstract contract ETHLockbox_TestInit is CommonTest {
    error InvalidInitialization();

    event ETHLocked(IOptimismPortal2 indexed portal, uint256 amount);
    event ETHUnlocked(IOptimismPortal2 indexed portal, uint256 amount);
    event PortalAuthorized(IOptimismPortal2 indexed portal);
    event LockboxAuthorized(IETHLockbox indexed lockbox);
    event LiquidityMigrated(IETHLockbox indexed lockbox, uint256 amount);
    event LiquidityReceived(IETHLockbox indexed lockbox, uint256 amount);

    function setUp() public virtual override {
        super.setUp();

        // If the ETHLockbox system feature is not enabled, skip these tests.
        skipIfSysFeatureDisabled(Features.ETH_LOCKBOX);
    }
}

/// @title ETHLockbox_Version_Test
/// @notice Test contract for the `version` function.
contract ETHLockbox_Version_Test is ETHLockbox_TestInit {
    /// @notice Tests that the `version` function returns a valid string. We avoid testing the
    ///         specific value of the string as it changes frequently.
    function test_version_succeeds() public view {
        assert(bytes(ethLockbox.version()).length > 0);
    }
}

/// @title ETHLockbox_Initialize_Test
/// @notice Test contract for the initialize function.
contract ETHLockbox_Initialize_Test is ETHLockbox_TestInit {
    /// @notice Tests the superchain config was correctly set during initialization.
    function test_initialize_succeeds() public view {
        assertEq(address(ethLockbox.systemConfig().superchainConfig()), address(superchainConfig));
        assertEq(ethLockbox.authorizedPortals(optimismPortal2), true);
        assertEq(address(ethLockbox.superchainConfig()), address(superchainConfig));
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal initialization
    ///         but confirms that the initValue is not incremented incorrectly if an upgrade
    ///         function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("ETHLockbox", "_initialized");

        // Get the initializer value.
        bytes32 slotVal = vm.load(address(ethLockbox), bytes32(slot.slot));
        uint8 val = uint8(uint256(slotVal) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, ethLockbox.initVersion());
    }

    /// @notice Tests that the `initialize` function reverts if called by a non-proxy admin or
    ///         owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("ETHLockbox", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(ethLockbox), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender
        vm.prank(_sender);
        IOptimismPortal2[] memory _portals = new IOptimismPortal2[](1);
        ethLockbox.initialize(systemConfig, _portals);
    }

    /// @notice Tests it reverts when the contract is already initialized.
    function test_initialize_alreadyInitialized_reverts() public {
        vm.expectRevert("Initializable: contract is already initialized");
        IOptimismPortal2[] memory _portals = new IOptimismPortal2[](1);
        ethLockbox.initialize(systemConfig, _portals);
    }
}

/// @title ETHLockbox_Paused_Test
/// @notice Test contract for the `paused` function.
contract ETHLockbox_Paused_Test is ETHLockbox_TestInit {
    /// @notice Tests the `paused` status is correctly returned.
    function test_paused_succeeds() public {
        // Assert the paused status is false
        assertEq(ethLockbox.paused(), false);

        // Mock the superchain config to return true for the paused status
        // We use abi.encodeWithSignature because paused is overloaded.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(superchainConfig), abi.encodeWithSignature("paused(address)", address(0)), abi.encode(true));

        // Assert the paused status is true
        assertEq(ethLockbox.paused(), true);
    }
}

/// @title ETHLockbox_AuthorizePortal_Test
/// @notice Test contract for the authorizePortal function.
contract ETHLockbox_AuthorizePortal_Test is ETHLockbox_TestInit {
    /// @notice Tests the `authorizePortal` function reverts when the caller is not the proxy
    ///         admin.
    function testFuzz_authorizePortal_unauthorized_reverts(address _caller) public {
        vm.assume(_caller != proxyAdminOwner);

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);

        // Call the `authorizePortal` function with an unauthorized caller
        vm.prank(_caller);
        ethLockbox.authorizePortal(optimismPortal2);
    }

    /// @notice Tests the `authorizePortal` function reverts when the proxy admin owner of the
    ///         portal is not the same as the one of the lockbox.
    function testFuzz_authorizePortal_differentProxyAdminOwner_reverts(IOptimismPortal2 _portal) public {
        assumeNotForgeAddress(address(_portal));
        vm.mockCall(address(_portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0)));

        // Expect the revert with `DifferentOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotSharedProxyAdminOwner.selector);

        // Call the `authorizePortal` function
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizePortal(_portal);
    }

    /// @notice Tests the `authorizePortal` function reverts when the portal has a different
    ///         SuperchainConfig than the one configured in the lockbox.
    /// @param _portal The portal to authorize.
    function testFuzz_authorizePortal_differentSuperchainConfig_reverts(IOptimismPortal2 _portal) public {
        assumeNotForgeAddress(address(_portal));
        vm.assume(address(_portal) != address(systemConfig));
        vm.assume(address(_portal) != EIP1967Helper.getImplementation(address(systemConfig)));

        // Mock the portal to have the right proxyAdminOwner.
        vm.mockCall(
            address(_portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Mock the portal to have the wrong SuperchainConfig.
        vm.mockCall(address(_portal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(address(0)));

        // Expect the revert with `DifferentSuperchainConfig` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_DifferentSuperchainConfig.selector);

        // Call the `authorizePortal` function
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizePortal(_portal);
    }

    /// @notice Tests the `authorizePortal` function succeeds using the `optimismPortal2` address
    ///         as the portal.
    function test_authorizePortal_succeeds() public {
        // Calculate the correct storage slot for the mapping value
        bytes32 mappingSlot = bytes32(uint256(1)); // position on the layout
        address key = address(optimismPortal2);
        bytes32 slot = keccak256(abi.encode(key, mappingSlot));

        // Reset the authorization status to false
        vm.store(address(ethLockbox), slot, bytes32(0));

        // Expect the `PortalAuthorized` event to be emitted
        vm.expectEmit(address(ethLockbox));
        emit PortalAuthorized(optimismPortal2);

        // Call the `authorizePortal` function with the portal
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizePortal(optimismPortal2);

        // Assert the portal is authorized
        assertTrue(ethLockbox.authorizedPortals(optimismPortal2));
    }

    /// @notice Tests the `authorizeLockbox` function succeeds
    function testFuzz_authorizePortal_succeeds(IOptimismPortal2 _portal) public {
        assumeNotForgeAddress(address(_portal));

        // Mock the admin owner of the portal to be the same as the current lockbox proxy admin
        // owner
        vm.mockCall(
            address(_portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Mock the SuperchainConfig on the portal to be the same as the SuperchainConfig on the
        // Lockbox.
        vm.mockCall(
            address(_portal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
        );

        // Expect the `PortalAuthorized` event to be emitted
        vm.expectEmit(address(ethLockbox));
        emit PortalAuthorized(_portal);

        // Call the `authorizePortal` function with the portal
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizePortal(_portal);

        // Assert the portal is authorized
        assertTrue(ethLockbox.authorizedPortals(_portal));
    }
}

/// @title ETHLockbox_ReceiveLiquidity_Test
/// @notice Test contract for the receiveLiquidity function.
contract ETHLockbox_ReceiveLiquidity_Test is ETHLockbox_TestInit {
    /// @notice Tests the liquidity is correctly received.
    function testFuzz_receiveLiquidity_succeeds(address _lockbox, uint256 _value) public {
        // Since on the fork the `_lockbox` fuzzed address doesn't exist, we skip the test
        if (isL1ForkTest()) vm.skip(true);
        assumeNotForgeAddress(_lockbox);
        vm.assume(address(_lockbox) != address(ethLockbox));

        // Deal the value to the lockbox
        deal(address(_lockbox), _value);

        // Mock the admin owner of the lockbox to be the same as the current lockbox proxy admin
        // owner
        vm.mockCall(
            address(_lockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Authorize the lockbox if needed
        if (!ethLockbox.authorizedLockboxes(IETHLockbox(_lockbox))) {
            vm.prank(proxyAdminOwner);
            ethLockbox.authorizeLockbox(IETHLockbox(_lockbox));
        }

        // Get the balance of the lockbox before the receive
        uint256 ethLockboxBalanceBefore = address(ethLockbox).balance;

        // Expect the `LiquidityReceived` event to be emitted
        vm.expectEmit(address(ethLockbox));
        emit LiquidityReceived(IETHLockbox(_lockbox), _value);

        // Call the `receiveLiquidity` function
        vm.prank(address(_lockbox));
        ethLockbox.receiveLiquidity{ value: _value }();

        // Assert the lockbox's balance increased by the amount received
        assertEq(address(ethLockbox).balance, ethLockboxBalanceBefore + _value);
    }
}

/// @title ETHLockbox_LockETH_Test
/// @notice Test contract for the lockETH function.
contract ETHLockbox_LockETH_Test is ETHLockbox_TestInit {
    /// @notice Tests it reverts when the caller is not an authorized portal.
    function testFuzz_lockETH_unauthorizedPortal_reverts(address _caller) public {
        vm.assume(!ethLockbox.authorizedPortals(IOptimismPortal2(payable(_caller))));

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_Unauthorized.selector);

        // Call the `lockETH` function with an unauthorized caller
        vm.prank(_caller);
        ethLockbox.lockETH();
    }

    /// @notice Tests the ETH is correctly locked when the caller is an authorized portal.
    function testFuzz_lockETH_succeeds(uint256 _amount) public {
        // Prevent overflow on an upgrade context
        _amount = bound(_amount, 0, type(uint256).max - address(ethLockbox).balance);

        // Deal the ETH amount to the portal
        vm.deal(address(optimismPortal2), _amount);

        // Get the balance of the portal and lockbox before the lock to compare later on the
        // assertions
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // Look for the emit of the `ETHLocked` event
        vm.expectEmit(address(ethLockbox));
        emit ETHLocked(optimismPortal2, _amount);

        // Call the `lockETH` function with the portal
        vm.prank(address(optimismPortal2));
        ethLockbox.lockETH{ value: _amount }();

        // Assert the portal's balance decreased and the lockbox's balance increased by the
        // amount locked
        assertEq(address(optimismPortal2).balance, portalBalanceBefore - _amount);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore + _amount);
    }

    /// @notice Tests the ETH is correctly locked when the caller is an authorized portal with
    ///         different portals.
    function testFuzz_lockETH_multiplePortals_succeeds(IOptimismPortal2 _portal, uint256 _amount) public {
        // Since on the fork the `_portal` fuzzed address doesn't exist, we skip the test
        if (isL1ForkTest()) vm.skip(true);
        assumeNotForgeAddress(address(_portal));
        vm.assume(address(_portal) != address(ethLockbox));

        // Mock the admin owner of the portal to be the same as the current lockbox proxy admin
        // owner
        vm.mockCall(
            address(_portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Mock the SuperchainConfig on the portal to be the same as the SuperchainConfig on the
        // lockbox.
        vm.mockCall(
            address(_portal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
        );

        // Set the portal as an authorized portal if needed
        if (!ethLockbox.authorizedPortals(_portal)) {
            vm.prank(proxyAdminOwner);
            ethLockbox.authorizePortal(_portal);
        }

        // Deal the ETH amount to the portal
        vm.deal(address(_portal), _amount);

        // Get the balance of the lockbox before the lock to compare later on the assertions
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // Look for the emit of the `ETHLocked` event
        vm.expectEmit(address(ethLockbox));
        emit ETHLocked(_portal, _amount);

        // Call the `lockETH` function with the portal
        vm.prank(address(_portal));
        ethLockbox.lockETH{ value: _amount }();

        // Assert the portal's balance decreased and the lockbox's balance increased by the
        // amount locked
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore + _amount);
    }
}

/// @title ETHLockbox_WithdrawalThrottle_TestInit
/// @notice Reusable initialization for ETHLockbox withdrawal throttle tests.
abstract contract ETHLockbox_WithdrawalThrottle_TestInit is ETHLockbox_TestInit {
    uint16 internal constant MAX_BPS = 1000;
    uint64 internal constant REFILL_PERIOD = 100;
    uint256 internal constant STOCK = 1000;
    uint256 internal constant CAPACITY = 100;

    /// @notice Sets a deterministic ETH stock for withdrawal throttle tests.
    function setUp() public virtual override {
        CommonTest.setUp();
        if (address(ethLockbox) == address(0)) {
            vm.skip(true, "Skipping test because ETHLockbox is not deployed");
        }
        vm.deal(address(ethLockbox), STOCK);
    }

    /// @notice Configures the lockbox withdrawal throttle.
    function _configureWithdrawalThrottle() internal {
        vm.prank(proxyAdminOwner);
        ethLockbox.setWithdrawalThrottle(MAX_BPS, REFILL_PERIOD);
    }
}

/// @title ETHLockbox_SetWithdrawalThrottle_Test
/// @notice Tests the `setWithdrawalThrottle` function.
contract ETHLockbox_SetWithdrawalThrottle_Test is ETHLockbox_WithdrawalThrottle_TestInit {
    /// @notice Tests that configuration snapshots the shared ETH stock and starts full.
    function test_setWithdrawalThrottle_succeeds() external {
        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleConfigured(MAX_BPS, REFILL_PERIOD, STOCK, CAPACITY, CAPACITY);
        _configureWithdrawalThrottle();

        IETHLockbox.WithdrawalThrottleConfig memory throttle = ethLockbox.withdrawalThrottle();
        assertEq(throttle.capacity, CAPACITY);
        assertEq(throttle.available, CAPACITY);
        assertEq(throttle.refillPeriod, REFILL_PERIOD);
        assertEq(throttle.lastUpdated, block.timestamp);
        assertEq(throttle.refillRemainder, 0);
        assertEq(throttle.maxBps, MAX_BPS);
        assertTrue(throttle.enabled);
    }

    /// @notice Tests that only the ProxyAdmin owner can configure the throttle.
    function testFuzz_setWithdrawalThrottle_notProxyAdminOwner_reverts(address _caller) external {
        vm.assume(_caller != proxyAdminOwner);

        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);
        vm.prank(_caller);
        ethLockbox.setWithdrawalThrottle(MAX_BPS, REFILL_PERIOD);
    }

    /// @notice Tests that zero basis points revert.
    function test_setWithdrawalThrottle_zeroBps_reverts() external {
        vm.expectRevert(IETHLockbox.ETHLockbox_InvalidWithdrawalThrottleBps.selector);
        vm.prank(proxyAdminOwner);
        ethLockbox.setWithdrawalThrottle(0, REFILL_PERIOD);
    }

    /// @notice Tests that basis points above 100% revert.
    function test_setWithdrawalThrottle_bpsAboveMaximum_reverts() external {
        vm.expectRevert(IETHLockbox.ETHLockbox_InvalidWithdrawalThrottleBps.selector);
        vm.prank(proxyAdminOwner);
        ethLockbox.setWithdrawalThrottle(10_001, REFILL_PERIOD);
    }

    /// @notice Tests that a zero refill period reverts.
    function test_setWithdrawalThrottle_zeroRefillPeriod_reverts() external {
        vm.expectRevert(IETHLockbox.ETHLockbox_InvalidWithdrawalThrottlePeriod.selector);
        vm.prank(proxyAdminOwner);
        ethLockbox.setWithdrawalThrottle(MAX_BPS, 0);
    }
}

/// @title ETHLockbox_RefreshWithdrawalThrottle_Test
/// @notice Tests the `refreshWithdrawalThrottle` function.
contract ETHLockbox_RefreshWithdrawalThrottle_Test is ETHLockbox_WithdrawalThrottle_TestInit {
    /// @notice Tests that refreshing increased stock preserves rather than refills availability.
    function test_refreshWithdrawalThrottle_preservesAvailable_succeeds() external {
        _configureWithdrawalThrottle();
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(60);
        vm.deal(address(ethLockbox), 2000);

        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleRefreshed(2000, 200, 40);
        vm.prank(proxyAdminOwner);
        ethLockbox.refreshWithdrawalThrottle();

        IETHLockbox.WithdrawalThrottleConfig memory throttle = ethLockbox.withdrawalThrottle();
        assertEq(throttle.capacity, 200);
        assertEq(throttle.available, 40);
    }
}

/// @title ETHLockbox_DisableWithdrawalThrottle_Test
/// @notice Tests the `disableWithdrawalThrottle` function.
contract ETHLockbox_DisableWithdrawalThrottle_Test is ETHLockbox_WithdrawalThrottle_TestInit {
    /// @notice Tests that disabling restores unrestricted ETH unlocks.
    function test_disableWithdrawalThrottle_succeeds() external {
        _configureWithdrawalThrottle();

        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleDisabled();
        vm.prank(proxyAdminOwner);
        ethLockbox.disableWithdrawalThrottle();

        assertEq(ethLockbox.availableWithdrawalCapacity(), type(uint256).max);
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(CAPACITY + 1);
    }
}

/// @title ETHLockbox_UnlockETH_Test
/// @notice Test contract for the unlockETH function.
contract ETHLockbox_UnlockETH_Test is ETHLockbox_WithdrawalThrottle_TestInit {
    /// @notice Tests `unlockETH` reverts when the contract is paused.
    function testFuzz_unlockETH_paused_reverts(address _caller, uint256 _value) public {
        // Mock the superchain config to return true for the paused status
        // We use abi.encodeWithSignature because paused is overloaded.
        // nosemgrep: sol-style-use-abi-encodecall
        vm.mockCall(address(superchainConfig), abi.encodeWithSignature("paused(address)", address(0)), abi.encode(true));

        // Expect the revert with `Paused` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_Paused.selector);

        // Call the `unlockETH` function with the caller
        vm.prank(_caller);
        ethLockbox.unlockETH(_value);
    }

    /// @notice Tests it reverts when the caller is not an authorized portal.
    function testFuzz_unlockETH_unauthorizedPortal_reverts(address _caller, uint256 _value) public {
        vm.assume(!ethLockbox.authorizedPortals(IOptimismPortal2(payable(_caller))));

        // Expect the revert with `Unauthorized` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_Unauthorized.selector);

        // Call the `unlockETH` function with an unauthorized caller
        vm.prank(_caller);
        ethLockbox.unlockETH(_value);
    }

    /// @notice Tests `unlockETH` reverts when the `_value` input is greater than the balance of
    ///         the lockbox.
    function testFuzz_unlockETH_insufficientBalance_reverts(uint256 _value) public {
        _value = bound(_value, address(ethLockbox).balance + 1, type(uint256).max);

        // Expect the revert with `InsufficientBalance` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_InsufficientBalance.selector);

        // Call the `unlockETH` function with the portal
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(_value);
    }

    /// @notice Tests `unlockETH` reverts when the portal is not the L2 sender to prevent
    ///         unlocking ETH from the lockbox through a withdrawal transaction.
    function testFuzz_unlockETH_withdrawalTransaction_reverts(uint256 _value, address _l2Sender) public {
        _value = bound(_value, 0, address(ethLockbox).balance);
        vm.assume(_l2Sender != Constants.DEFAULT_L2_SENDER);

        // Mock the L2 sender
        vm.mockCall(address(optimismPortal2), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(_l2Sender));

        // Expect the revert with `NoWithdrawalTransactions` selector
        vm.expectRevert(IETHLockbox.ETHLockbox_NoWithdrawalTransactions.selector);

        // Call the `unlockETH` function with the portal
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(_value);
    }

    /// @notice Tests the ETH is correctly unlocked when the caller is an authorized portal.
    function testFuzz_unlockETH_succeeds(uint256 _value) public {
        // Deal the ETH amount to the lockbox
        vm.deal(address(ethLockbox), _value);

        // Get the balance of the portal and lockbox before the unlock to compare later on the
        // assertions
        uint256 portalBalanceBefore = address(optimismPortal2).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // Expect `donateETH` function to be called on Portal
        vm.expectCall(address(optimismPortal2), abi.encodeCall(IOptimismPortal2.donateETH, ()));

        // Look for the emit of the `ETHUnlocked` event
        vm.expectEmit(address(ethLockbox));
        emit ETHUnlocked(optimismPortal2, _value);

        // Call the `unlockETH` function with the portal
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(_value);

        // Assert the portal's balance increased and the lockbox's balance decreased by the amount
        // unlocked
        assertEq(address(optimismPortal2).balance, portalBalanceBefore + _value);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore - _value);
    }

    /// @notice Tests the ETH is correctly unlocked when the caller is an authorized portal.
    function testFuzz_unlockETH_multiplePortals_succeeds(IOptimismPortal2 _portal, uint256 _value) public {
        assumeNotForgeAddress(address(_portal));

        // Mock the admin owner of the portal to be the same as the current lockbox proxy admin
        // owner
        vm.mockCall(
            address(_portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Mock the SuperchainConfig on the portal to be the same as the SuperchainConfig on the
        // lockbox.

        vm.mockCall(
            address(_portal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
        );
        vm.mockCall(
            address(_portal), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(Constants.DEFAULT_L2_SENDER)
        );

        // Set the portal as an authorized portal if needed
        if (!ethLockbox.authorizedPortals(_portal)) {
            vm.prank(proxyAdminOwner);
            ethLockbox.authorizePortal(_portal);
        }

        // Deal the ETH amount to the lockbox
        vm.deal(address(ethLockbox), _value);

        // Get the balance of the portal and lockbox before the unlock to compare later on the
        // assertions
        uint256 portalBalanceBefore = address(_portal).balance;
        uint256 lockboxBalanceBefore = address(ethLockbox).balance;

        // Expect `donateETH` function to be called on Portal
        vm.expectCall(address(_portal), abi.encodeCall(IOptimismPortal2.donateETH, ()));

        // Look for the emit of the `ETHUnlocked` event
        vm.expectEmit(address(ethLockbox));
        emit ETHUnlocked(_portal, _value);

        // Call the `unlockETH` function with the portal
        vm.prank(address(_portal));
        ethLockbox.unlockETH(_value);

        // Assert the portal's balance increased and the lockbox's balance decreased by the amount
        // unlocked
        assertEq(address(_portal).balance, portalBalanceBefore + _value);
        assertEq(address(ethLockbox).balance, lockboxBalanceBefore - _value);
    }

    /// @notice Tests that authorized portals consume one shared withdrawal bucket.
    function test_unlockETH_multiplePortalsConsumeSharedCapacity_succeeds() external {
        IOptimismPortal2 portal = IOptimismPortal2(payable(address(0xBEEF)));
        vm.mockCall(
            address(portal), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );
        vm.mockCall(
            address(portal), abi.encodeCall(IOptimismPortal2.superchainConfig, ()), abi.encode(superchainConfig)
        );
        vm.mockCall(
            address(portal), abi.encodeCall(IOptimismPortal2.l2Sender, ()), abi.encode(Constants.DEFAULT_L2_SENDER)
        );
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizePortal(portal);
        _configureWithdrawalThrottle();

        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(40);
        vm.prank(address(portal));
        ethLockbox.unlockETH(60);

        assertEq(ethLockbox.availableWithdrawalCapacity(), 0);
    }

    /// @notice Tests that consuming exact capacity succeeds and exhausts the shared bucket.
    function test_unlockETH_exactWithdrawalCapacity_succeeds() external {
        _configureWithdrawalThrottle();

        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleCapacityConsumed(CAPACITY, 0);
        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleCapacityExhausted();
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(CAPACITY);

        assertEq(ethLockbox.availableWithdrawalCapacity(), 0);
    }

    /// @notice Tests that an unlock reverts when the shared bucket lacks capacity.
    function test_unlockETH_insufficientWithdrawalCapacity_reverts() external {
        _configureWithdrawalThrottle();
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(CAPACITY);

        vm.expectRevert(abi.encodeWithSelector(IETHLockbox.ETHLockbox_WithdrawalThrottled.selector, 1, 0, 90));
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(1);
    }

    /// @notice Tests that increased stock raises the ceiling without immediately topping up availability.
    function test_unlockETH_increasedStockLazilyRaisesCapacity_succeeds() external {
        _configureWithdrawalThrottle();
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(CAPACITY);

        vm.deal(address(ethLockbox), 1900);
        assertEq(ethLockbox.availableWithdrawalCapacity(), 0);

        vm.warp(block.timestamp + REFILL_PERIOD / 2);
        assertEq(ethLockbox.availableWithdrawalCapacity(), 50);
        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleRefreshed(1900, 190, 50);
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(50);

        assertEq(ethLockbox.withdrawalThrottle().capacity, 190);
        vm.warp(block.timestamp + REFILL_PERIOD / 2);
        assertEq(ethLockbox.availableWithdrawalCapacity(), 95);
    }

    /// @notice Tests that decreased stock immediately clamps effective availability.
    function test_unlockETH_decreasedStockLazilyClampsCapacity_succeeds() external {
        _configureWithdrawalThrottle();
        vm.deal(address(ethLockbox), 500);

        assertEq(ethLockbox.availableWithdrawalCapacity(), 50);
        vm.expectEmit(address(ethLockbox));
        emit WithdrawalThrottleRefreshed(500, 50, 50);
        vm.prank(address(optimismPortal2));
        ethLockbox.unlockETH(50);

        assertEq(ethLockbox.withdrawalThrottle().capacity, 50);
    }
}

/// @title ETHLockbox_AuthorizeLockbox_Test
/// @notice Test contract for the authorizeLockbox function.
contract ETHLockbox_AuthorizeLockbox_Test is ETHLockbox_TestInit {
    /// @notice Tests the `authorizeLockbox` function reverts when the caller is not the proxy
    ///         admin.
    function testFuzz_authorizeLockbox_unauthorized_reverts(address _caller) public {
        vm.assume(_caller != proxyAdminOwner);

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);

        // Call the `authorizeLockbox` function with an unauthorized caller
        vm.prank(_caller);
        ethLockbox.authorizeLockbox(ethLockbox);
    }

    /// @notice Tests the `authorizeLockbox` function reverts when the proxy admin owner of the
    ///         lockbox is not the same as the proxy admin owner of the proxy admin.
    function testFuzz_authorizeLockbox_differentProxyAdminOwner_reverts(address _lockbox) public {
        assumeNotForgeAddress(_lockbox);

        vm.mockCall(address(_lockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0)));

        // Expect the revert with `NotSharedProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotSharedProxyAdminOwner.selector);

        // Call the `authorizeLockbox` function with the lockbox
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizeLockbox(IETHLockbox(_lockbox));
    }

    /// @notice Tests the `authorizeLockbox` function succeeds
    function testFuzz_authorizeLockbox_succeeds(address _lockbox) public {
        assumeNotForgeAddress(_lockbox);

        // Mock the admin owner of the lockbox to be the same as the current lockbox proxy admin
        // owner
        vm.mockCall(
            address(_lockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(proxyAdminOwner)
        );

        // Expect the `LockboxAuthorized` event to be emitted
        vm.expectEmit(address(ethLockbox));
        emit LockboxAuthorized(IETHLockbox(_lockbox));

        // Authorize the lockbox
        vm.prank(proxyAdminOwner);
        ethLockbox.authorizeLockbox(IETHLockbox(_lockbox));

        // Assert the lockbox is authorized
        assertTrue(ethLockbox.authorizedLockboxes(IETHLockbox(_lockbox)));
    }
}

/// @title ETHLockbox_MigrateLiquidity_Test
/// @notice Test contract for the migrateLiquidity function.
contract ETHLockbox_MigrateLiquidity_Test is ETHLockbox_TestInit {
    /// @notice Tests the `migrateLiquidity` function reverts when the caller is not the proxy
    ///         admin.
    function testFuzz_migrateLiquidity_unauthorized_reverts(address _caller) public {
        vm.assume(_caller != proxyAdminOwner);

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOwner.selector);

        // Call the `migrateLiquidity` function with an unauthorized caller
        vm.prank(_caller);
        ethLockbox.migrateLiquidity(ethLockbox);
    }

    /// @notice Tests the `migrateLiquidity` function reverts when the proxy admin owner of the
    ///         lockbox is not the same as the proxy admin owner of the proxy admin.
    function testFuzz_migrateLiquidity_differentProxyAdminOwner_reverts(address _lockbox) public {
        assumeNotForgeAddress(_lockbox);

        vm.mockCall(address(_lockbox), abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()), abi.encode(address(0)));

        // Expect the revert with `NotSharedProxyAdminOwner` selector
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotSharedProxyAdminOwner.selector);

        // Call the `migrateLiquidity` function with the lockbox
        vm.prank(proxyAdminOwner);
        ethLockbox.migrateLiquidity(IETHLockbox(_lockbox));
    }

    /// @notice Tests the `migrateLiquidity` function succeeds
    function testFuzz_migrateLiquidity_succeeds(
        uint256 _originLockboxBalance,
        uint256 _destinationLockboxBalance
    )
        public
    {
        // Since on the fork the `_lockbox` fuzzed address doesn't exist, we skip the test
        if (isL1ForkTest()) vm.skip(true);

        // Bound balances to avoid overflow
        _originLockboxBalance = bound(_originLockboxBalance, 0, type(uint256).max - address(ethLockbox).balance);
        _destinationLockboxBalance = bound(_destinationLockboxBalance, 0, type(uint256).max - _originLockboxBalance);

        // Deploy a new Proxy for the destination lockbox
        address destinationLockbox = address(new Proxy(address(proxyAdmin)));

        // Get the ETHLockbox implementation of the origin `ethLockbox` proxy
        vm.prank(address(proxyAdmin));
        address implementation = Proxy(payable(address(ethLockbox))).implementation();

        // Upgrade the destination lockbox proxy to the `ETHLockbox` implementation
        vm.prank(address(proxyAdmin));
        Proxy(payable(destinationLockbox)).upgradeTo(implementation);

        // Authorize the origin lockbox on the destination lockbox
        vm.prank(proxyAdminOwner);
        IETHLockbox(destinationLockbox).authorizeLockbox(ethLockbox);

        // Mock the calls for checks on the destination lockbox so it can receive the migration
        vm.mockCall(
            address(destinationLockbox),
            abi.encodeCall(IProxyAdminOwnedBase.proxyAdminOwner, ()),
            abi.encode(proxyAdminOwner)
        );
        vm.mockCall(
            address(destinationLockbox), abi.encodeCall(IETHLockbox.authorizedLockboxes, (ethLockbox)), abi.encode(true)
        );

        // Deal the balance to both lockboxes
        deal(address(ethLockbox), _originLockboxBalance);
        deal(address(destinationLockbox), _destinationLockboxBalance);

        // Get balances before the migration
        uint256 originLockboxBalanceBefore = address(ethLockbox).balance;
        uint256 destLockboxBalanceBefore = address(destinationLockbox).balance;

        // Expect the `LiquidityMigrated` event to be emitted
        vm.expectEmit(address(ethLockbox));
        emit LiquidityMigrated(IETHLockbox(destinationLockbox), originLockboxBalanceBefore);

        // Call the `migrateLiquidity` function with the lockbox
        vm.prank(proxyAdminOwner);
        ethLockbox.migrateLiquidity(IETHLockbox(destinationLockbox));

        // Assert the liquidity was migrated
        assertEq(address(ethLockbox).balance, 0);
        assertEq(address(destinationLockbox).balance, destLockboxBalanceBefore + originLockboxBalanceBefore);
    }
}

/// @title ETHLockbox_Uncategorized_Test
/// @notice Contains uncategorized tests related to ETHLockbox.
contract ETHLockbox_Uncategorized_Test is ETHLockbox_TestInit {
    /// @notice Tests the proxy admin owner is correctly returned.
    function test_proxyProxyAdminOwner_succeeds() public view {
        assertEq(ethLockbox.proxyAdminOwner(), proxyAdminOwner);
    }
}
