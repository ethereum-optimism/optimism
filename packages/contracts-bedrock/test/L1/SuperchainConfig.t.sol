// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Libraries
import { DeployUtils } from "scripts/libraries/DeployUtils.sol";
import { ForgeArtifacts, StorageSlot } from "scripts/libraries/ForgeArtifacts.sol";
import { Constants } from "src/libraries/Constants.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";
import { ISuperchainConfig } from "interfaces/L1/ISuperchainConfig.sol";
import { IProxyAdminOwnedBase } from "interfaces/L1/IProxyAdminOwnedBase.sol";

/// @title SuperchainConfig_TestInit
/// @notice Initialization contract for SuperchainConfig tests.
contract SuperchainConfig_TestInit is CommonTest {
    function setUp() public virtual override {
        super.setUp();
        skipIfForkTest("SuperchainConfig_TestInit: cannot test initialization on forked network");
    }
}

/// @title SuperchainConfig_Initialize_Test
/// @notice Test contract for SuperchainConfig `initialize` function.
contract SuperchainConfig_Initialize_Test is SuperchainConfig_TestInit {
    /// @notice Tests that initialization sets the correct values. These are defined in
    ///         CommonTest.sol.
    function test_initialize_unpaused_succeeds() external view {
        assertFalse(superchainConfig.paused(address(this)));
        assertEq(superchainConfig.guardian(), deploy.cfg().superchainConfigGuardian());
    }

    /// @notice Tests that it can be initialized as paused.
    function test_initialize_paused_succeeds() external {
        IProxy newProxy = IProxy(
            DeployUtils.create1({
                _name: "Proxy",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(IProxy.__constructor__, (alice)))
            })
        );
        ISuperchainConfig newImpl = ISuperchainConfig(
            DeployUtils.create1({
                _name: "SuperchainConfig",
                _args: DeployUtils.encodeConstructor(abi.encodeCall(ISuperchainConfig.__constructor__, ()))
            })
        );

        vm.startPrank(alice);
        newProxy.upgradeToAndCall(
            address(newImpl), abi.encodeCall(ISuperchainConfig.initialize, (deploy.cfg().superchainConfigGuardian()))
        );

        assertFalse(ISuperchainConfig(address(newProxy)).paused(address(this)));
        assertEq(ISuperchainConfig(address(newProxy)).guardian(), deploy.cfg().superchainConfigGuardian());
    }

    /// @notice Tests that the initializer value is correct. Trivial test for normal
    ///         initialization but confirms that the initValue is not incremented incorrectly if
    ///         an upgrade function is not present.
    function test_initialize_correctInitializerValue_succeeds() public {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("SuperchainConfig", "_initialized");

        // Get the initializer value.
        bytes32 slotVal = vm.load(address(superchainConfig), bytes32(slot.slot));
        uint8 val = uint8(uint256(slotVal) & 0xFF);

        // Assert that the initializer value matches the expected value.
        assertEq(val, superchainConfig.initVersion());
    }

    /// @notice Tests that the `initialize` function reverts if called by a non-proxy admin or
    ///         owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_initialize_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(proxyAdmin) && _sender != proxyAdminOwner);

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("SuperchainConfig", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(superchainConfig), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector.
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `initialize` function with the sender
        vm.prank(_sender);
        superchainConfig.initialize(address(0xdeadbeef));
    }
}

/// @title SuperchainConfig_Upgrade_Test
/// @notice Test contract for SuperchainConfig `upgrade` function.
contract SuperchainConfig_Upgrade_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `upgrade` successfully upgrades the contract.
    function test_upgrade_succeeds() external {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("SuperchainConfig", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(superchainConfig), bytes32(slot.slot), bytes32(0));

        // Get the slot for the SuperchainConfig's ProxyAdmin.
        address proxyAdminAddress =
            address(uint160(uint256(vm.load(address(superchainConfig), Constants.PROXY_OWNER_ADDRESS))));

        // Upgrade the contract.
        vm.prank(proxyAdminAddress);
        superchainConfig.upgrade();

        // Check that the guardian slot was updated.
        bytes32 guardianSlot = bytes32(uint256(keccak256("superchainConfig.guardian")) - 1);
        assertEq(vm.load(address(superchainConfig), guardianSlot), bytes32(0));

        // Check that the paused slot was cleared.
        bytes32 pausedSlot = bytes32(uint256(keccak256("superchainConfig.paused")) - 1);
        assertEq(vm.load(address(superchainConfig), pausedSlot), bytes32(0));
    }

    /// @notice Tests that `upgrade` reverts when called a second time.
    function test_upgrade_upgradeTwice_reverts() external {
        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("SuperchainConfig", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(superchainConfig), bytes32(slot.slot), bytes32(0));

        // Get the slot for the SuperchainConfig's ProxyAdmin.
        address proxyAdminAddress =
            address(uint160(uint256(vm.load(address(superchainConfig), Constants.PROXY_OWNER_ADDRESS))));

        // Trigger first upgrade.
        vm.prank(proxyAdminAddress);
        superchainConfig.upgrade();

        // Trigger second upgrade.
        vm.prank(proxyAdminAddress);
        vm.expectRevert("Initializable: contract is already initialized");
        superchainConfig.upgrade();
    }

    /// @notice Tests that `upgrade` reverts when called by a non-proxy admin or owner.
    /// @param _sender The address of the sender to test.
    function testFuzz_upgrade_notProxyAdminOrProxyAdminOwner_reverts(address _sender) public {
        // Prank as the not ProxyAdmin or ProxyAdmin owner.
        vm.assume(_sender != address(superchainProxyAdmin) && _sender != superchainProxyAdminOwner);

        // Get the slot for _initialized.
        StorageSlot memory slot = ForgeArtifacts.getSlot("SuperchainConfig", "_initialized");

        // Set the initialized slot to 0.
        vm.store(address(superchainConfig), bytes32(slot.slot), bytes32(0));

        // Expect the revert with `ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner` selector.
        vm.expectRevert(IProxyAdminOwnedBase.ProxyAdminOwnedBase_NotProxyAdminOrProxyAdminOwner.selector);

        // Call the `upgrade` function with the sender
        vm.prank(_sender);
        superchainConfig.upgrade();
    }
}

/// @title SuperchainConfig_PauseExpiry_Test
/// @notice Test contract for SuperchainConfig `pauseExpiry` function.
contract SuperchainConfig_PauseExpiry_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pauseExpiry` returns the correct constant value.
    function test_pauseExpiry_succeeds() external view {
        assertEq(superchainConfig.pauseExpiry(), 7_884_000);
    }
}

/// @title SuperchainConfig_Paused_Test
/// @notice Test contract for SuperchainConfig `paused` function.
contract SuperchainConfig_Paused_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `paused` returns true when the specific identifier is paused.
    /// @param _identifier The identifier to test.
    function testFuzz_paused_specificIdentifier_succeeds(address _identifier) external {
        // Assume the identifier is not the zero address.
        vm.assume(_identifier != address(0));

        // Pause with the specific identifier.
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);

        // Assert that the specific identifier is paused.
        assertTrue(superchainConfig.paused(_identifier));

        // Pick a random address that is not the identifier.
        address other = vm.randomAddress();
        while (other == _identifier) {
            other = vm.randomAddress();
        }

        // Assert that the other address is not paused.
        assertFalse(superchainConfig.paused(other));
    }

    /// @notice Tests that `paused` returns true when the global superchain system is paused.
    function test_paused_global_succeeds() external {
        // Pause the global superchain system.
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(0));

        // Assert that the global superchain system is paused.
        assertTrue(superchainConfig.paused());
        assertTrue(superchainConfig.paused(address(0)));

        // Pick a random address that is not the zero address.
        address other = vm.randomAddress();
        while (other == address(0)) {
            other = vm.randomAddress();
        }

        // Assert that the other address is not paused.
        assertFalse(superchainConfig.paused(other));
    }

    /// @notice Tests that `paused` returns false after pause expires.
    /// @param _identifier The identifier to test.
    function testFuzz_paused_expired_succeeds(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        assertTrue(superchainConfig.paused(_identifier));

        // Warp past expiry
        vm.warp(block.timestamp + superchainConfig.pauseExpiry() + 1);
        assertFalse(superchainConfig.paused(_identifier));
    }

    /// @notice Tests that `paused` returns true just before expiry.
    /// @param _identifier The identifier to test.
    function testFuzz_paused_beforeExpiry_succeeds(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);

        // Warp to just before expiry
        vm.warp(block.timestamp + superchainConfig.pauseExpiry() - 1);
        assertTrue(superchainConfig.paused(_identifier));
    }
}

/// @title SuperchainConfig_Pause_Test
/// @notice Test contract for SuperchainConfig `pause` function.
contract SuperchainConfig_Pause_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pause` successfully pauses when called by the guardian.
    /// @param _identifier The identifier to test.
    function testFuzz_pause_succeeds(address _identifier) external {
        assertFalse(superchainConfig.paused(_identifier));

        vm.expectEmit(address(superchainConfig));
        emit Paused(_identifier);

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);

        assertTrue(superchainConfig.paused(_identifier));
    }

    /// @notice Tests that `pause` reverts when called by a non-guardian.
    /// @param _caller The non-guardian caller to test.
    function testFuzz_pause_notGuardian_reverts(address _caller) external {
        vm.assume(_caller != superchainConfig.guardian());

        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        vm.prank(_caller);
        superchainConfig.pause(address(this));

        assertFalse(superchainConfig.paused(address(this)));
    }

    /// @notice Tests that `pause` reverts when the identifier is already used.
    /// @param _identifier The identifier to test.
    function testFuzz_pause_alreadyUsed_reverts(address _identifier) external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);

        vm.expectRevert(abi.encodeWithSelector(ISuperchainConfig.SuperchainConfig_AlreadyPaused.selector, _identifier));

        superchainConfig.pause(_identifier);
    }
}

/// @title SuperchainConfig_Unpause_Test
/// @notice Test contract for SuperchainConfig `unpause` function.
contract SuperchainConfig_Unpause_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `unpause` successfully unpauses when called by the guardian.
    /// @param _identifier The identifier to test.
    function testFuzz_unpause_succeeds(address _identifier) external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        assertTrue(superchainConfig.paused(_identifier));

        vm.expectEmit(address(superchainConfig));
        emit Unpaused(_identifier);
        superchainConfig.unpause(_identifier);

        assertFalse(superchainConfig.paused(_identifier));
    }

    /// @notice Tests that `unpause` reverts when called by a non-guardian.
    /// @param _caller The non-guardian caller to test.
    function testFuzz_unpause_notGuardian_reverts(address _caller) external {
        vm.assume(_caller != superchainConfig.guardian());

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        assertTrue(superchainConfig.paused(address(this)));

        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        vm.prank(_caller);
        superchainConfig.unpause(address(this));

        assertTrue(superchainConfig.paused(address(this)));
    }
}

/// @title SuperchainConfig_Extend_Test
/// @notice Test contract for SuperchainConfig `extend` function.
contract SuperchainConfig_Extend_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `extend` successfully resets and re-pauses an identifier.
    /// @param _identifier The identifier to test.
    function testFuzz_extend_succeeds(address _identifier) external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        uint256 firstPauseTimestamp = block.timestamp;

        vm.warp(block.timestamp + 1);

        superchainConfig.extend(_identifier);
        assertTrue(superchainConfig.pauseTimestamps(_identifier) > firstPauseTimestamp);
        assertTrue(superchainConfig.paused(_identifier));
    }

    /// @notice Tests that `extend` reverts when called by a non-guardian.
    /// @param _caller The non-guardian caller to test.
    function testFuzz_extend_notGuardian_reverts(address _caller) external {
        vm.assume(_caller != superchainConfig.guardian());

        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));

        vm.prank(_caller);
        vm.expectRevert(ISuperchainConfig.SuperchainConfig_OnlyGuardian.selector);
        superchainConfig.extend(address(this));
    }

    /// @notice Tests that `extend` reverts when the identifier is not already paused.
    /// @param _identifier The identifier to test.
    function testFuzz_extend_notAlreadyPaused_reverts(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        vm.expectRevert(
            abi.encodeWithSelector(ISuperchainConfig.SuperchainConfig_NotAlreadyPaused.selector, _identifier)
        );
        superchainConfig.extend(_identifier);
    }
}

/// @title SuperchainConfig_Pausable_Test
/// @notice Test contract for SuperchainConfig `pausable` function.
contract SuperchainConfig_Pausable_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pausable` returns true when the identifier is not paused.
    /// @param _identifier The identifier to test.
    function testFuzz_pausable_notPaused_succeeds(address _identifier) external view {
        assertTrue(superchainConfig.pausable(_identifier));
    }

    /// @notice Tests that `pausable` returns false when the identifier is paused.
    /// @param _identifier The identifier to test.
    function testFuzz_pausable_paused_succeeds(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        assertFalse(superchainConfig.pausable(_identifier));
    }

    /// @notice Tests that `pausable` returns false even after pause expires.
    /// @param _identifier The identifier to test.
    function testFuzz_pausable_expired_succeeds(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);

        // Warp past expiry
        vm.warp(block.timestamp + superchainConfig.pauseExpiry() + 1);

        // pausable() should still return false because timestamp is set
        assertFalse(superchainConfig.pausable(_identifier));
        // But paused() should return false because pause expired
        assertFalse(superchainConfig.paused(_identifier));
    }
}

/// @title SuperchainConfig_Guardian_Test
/// @notice Test contract for SuperchainConfig `guardian` getter function.
contract SuperchainConfig_Guardian_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `guardian` returns the correct guardian address.
    function test_guardian_succeeds() external view {
        assertEq(superchainConfig.guardian(), deploy.cfg().superchainConfigGuardian());
    }
}

/// @title SuperchainConfig_PauseTimestamps_Test
/// @notice Test contract for SuperchainConfig `pauseTimestamps` getter function.
contract SuperchainConfig_PauseTimestamps_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `pauseTimestamps` returns 0 for unpaused identifiers.
    /// @param _identifier The identifier to test.
    function testFuzz_pauseTimestamps_unpaused_succeeds(address _identifier) external view {
        assertEq(superchainConfig.pauseTimestamps(_identifier), 0);
    }

    /// @notice Tests that `pauseTimestamps` returns the correct timestamp for paused identifiers.
    /// @param _identifier The identifier to test.
    function testFuzz_pauseTimestamps_paused_succeeds(address _identifier) external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        assertEq(superchainConfig.pauseTimestamps(_identifier), block.timestamp);
    }

    /// @notice Tests that `pauseTimestamps` returns 0 after unpausing.
    /// @param _identifier The identifier to test.
    function testFuzz_pauseTimestamps_afterUnpause_succeeds(address _identifier) external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        assertTrue(superchainConfig.pauseTimestamps(_identifier) != 0);

        superchainConfig.unpause(_identifier);
        assertEq(superchainConfig.pauseTimestamps(_identifier), 0);
    }
}

/// @title SuperchainConfig_Version_Test
/// @notice Test contract for SuperchainConfig `version` getter function.
contract SuperchainConfig_Version_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `version` returns the correct version string.
    function test_version_succeeds() external view {
        assertEq(superchainConfig.version(), "2.3.0");
    }
}

/// @title SuperchainConfig_Expiration_Test
/// @notice Test contract for SuperchainConfig `expiration` function.
contract SuperchainConfig_Expiration_Test is SuperchainConfig_TestInit {
    /// @notice Tests that `expiration` returns 0 when the identifier is not paused.
    function test_expiration_notPaused_succeeds() external view {
        assertEq(superchainConfig.expiration(address(this)), 0);
    }

    /// @notice Tests that `expiration` returns the correct timestamp when the identifier is
    ///         paused.
    function test_expiration_paused_succeeds() external {
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        uint256 expectedExpiration = block.timestamp + superchainConfig.pauseExpiry();
        assertEq(superchainConfig.expiration(address(this)), expectedExpiration);
    }

    /// @notice Tests that `expiration` returns the updated timestamp after extending the pause.
    function test_expiration_afterExtend_succeeds() external {
        vm.startPrank(superchainConfig.guardian());
        superchainConfig.pause(address(this));
        uint256 firstExpiration = superchainConfig.expiration(address(this));

        // Warp forward in time
        vm.warp(block.timestamp + 100);

        // Extend the pause
        superchainConfig.extend(address(this));
        uint256 newExpiration = superchainConfig.expiration(address(this));

        assertTrue(newExpiration > firstExpiration);
        assertEq(newExpiration, block.timestamp + superchainConfig.pauseExpiry());
    }

    /// @notice Tests that `expiration` works correctly with fuzzed identifiers.
    /// @param _identifier The identifier to test.
    function testFuzz_expiration_succeeds(address _identifier) external {
        // Test unpaused state
        assertEq(superchainConfig.expiration(_identifier), 0);

        // Test paused state
        vm.prank(superchainConfig.guardian());
        superchainConfig.pause(_identifier);
        uint256 expectedExpiration = block.timestamp + superchainConfig.pauseExpiry();
        assertEq(superchainConfig.expiration(_identifier), expectedExpiration);
    }
}
