// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { SafeCompatibilityTest } from "test/safe-tools/SafeCompatibilityTest.sol";
import { Enum } from "safe-contracts/common/Enum.sol";
import { SaferSafes } from "src/safe/SaferSafes.sol";
import { LivenessModule2 } from "src/safe/LivenessModule2.sol";

/// @title SaferSafes_Uncategorized_Test
/// @notice Tests SaferSafes (module + guard) compatibility across all Safe versions
/// @dev This tests enabling and configuring SaferSafes as both a module and guard
///      across Safe versions v1.0.0 through v1.5.0 on Mainnet fork
contract SaferSafes_Uncategorized_Test is SafeCompatibilityTest {
    SaferSafes public saferSafes;
    address public fallbackOwner;

    // Configuration constants
    uint256 constant TIMELOCK_DELAY = 7 days;
    uint256 constant LIVENESS_RESPONSE_PERIOD = 21 days;

    // Guard was introduced in v1.3.0, so skip v1.0.0, v1.1.1, v1.2.0
    string[] public guardSkipVersions;

    function setUp() public override {
        // Initialize Safe versions on mainnet fork
        super.setUp();

        // Deploy SaferSafes contract
        saferSafes = new SaferSafes();

        // Set fallback owner
        fallbackOwner = makeAddr("fallbackOwner");

        // Initialize guard skip versions (guards introduced in v1.3.0)
        guardSkipVersions.push("v1.0.0");
        guardSkipVersions.push("v1.1.1");
        guardSkipVersions.push("v1.2.0");

        vm.label(address(saferSafes), "SaferSafes");
        vm.label(fallbackOwner, "FallbackOwner");
    }

    /// ============ STRICT MODE TESTS ============
    /// These tests will FAIL if any Safe version doesn't support the functionality

    /// @notice Test that SaferSafes can be enabled as a module on all versions
    function test_strict_enableModule_succeeds() public {
        forEachSafeVersion(this._testEnableModule);
    }

    /// @notice Test that SaferSafes can be enabled as a guard on all versions
    function test_strict_setGuard_succeeds() public {
        forEachSafeVersion(this._testSetGuard, guardSkipVersions);
    }

    /// @notice Test enabling both module and guard together
    function test_strict_enableBoth_succeeds() public {
        forEachSafeVersion(this._testEnableModuleAndGuard, guardSkipVersions);
    }

    /// @notice Test full configuration: enable module, enable guard, configure both
    function test_strict_fullConfiguration_succeeds() public {
        forEachSafeVersion(this._testFullConfiguration, guardSkipVersions);
    }

    /// ============ VERBOSE MODE TESTS ============
    /// These tests will LOG failures instead of reverting
    /// Use these to discover which versions support which features

    /// @notice VERBOSE: Discover which versions support SaferSafes as a module
    function test_verbose_enableModule_works() public {
        forEachSafeVersionVerbose(this._testEnableModule);
    }

    /// @notice VERBOSE: Discover which versions support SaferSafes as a guard
    function test_verbose_setGuard_works() public {
        forEachSafeVersionVerbose(this._testSetGuard, guardSkipVersions);
    }

    /// @notice VERBOSE: Discover which versions support both module and guard
    function test_verbose_enableBoth_works() public {
        forEachSafeVersionVerbose(this._testEnableModuleAndGuard, guardSkipVersions);
    }

    /// @notice VERBOSE: Discover which versions support full SaferSafes configuration
    function test_verbose_fullConfiguration_works() public {
        forEachSafeVersionVerbose(this._testFullConfiguration, guardSkipVersions);
    }

    /// @notice VERBOSE: Test configuration validation (liveness period must be >= 2x timelock)
    function test_verbose_configurationValidation_reverts() public {
        forEachSafeVersionVerbose(this._testConfigurationValidation, guardSkipVersions);
    }

    /// ============ IMPLEMENTATION FUNCTIONS ============

    /// @notice Implementation: Enable SaferSafes as a module
    function _testEnableModule(SafeVersion memory safeVersion) external {
        // Prepare transaction to enable SaferSafes as a module
        bytes memory data = abi.encodeCall(safeVersion.safe.enableModule, (address(saferSafes)));

        // Execute transaction
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, data, Enum.Operation.Call);

        // Verify module is enabled
        bool isEnabled = safeVersion.safe.isModuleEnabled(address(saferSafes));
        assertTrue(isEnabled, string.concat("SaferSafes module should be enabled on ", safeVersion.version));
    }

    /// @notice Implementation: Set SaferSafes as the guard
    function _testSetGuard(SafeVersion memory safeVersion) external {
        // Prepare transaction to set SaferSafes as guard
        bytes memory data = abi.encodeCall(safeVersion.safe.setGuard, (address(saferSafes)));

        // Execute transaction
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, data, Enum.Operation.Call);

        // Verify guard is set by checking storage
        bytes32 guardSlot = bytes32(uint256(keccak256("guard_manager.guard.address")) - 1);
        bytes32 guardValue = vm.load(address(safeVersion.safe), guardSlot);
        address currentGuard = address(uint160(uint256(guardValue)));

        assertEq(
            currentGuard, address(saferSafes), string.concat("SaferSafes guard should be set on ", safeVersion.version)
        );
    }

    /// @notice Implementation: Enable both module and guard
    function _testEnableModuleAndGuard(SafeVersion memory safeVersion) external {
        // Enable module
        bytes memory moduleData = abi.encodeCall(safeVersion.safe.enableModule, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, moduleData, Enum.Operation.Call);

        // Set guard
        bytes memory guardData = abi.encodeCall(safeVersion.safe.setGuard, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, guardData, Enum.Operation.Call);

        // Verify both are enabled
        assertTrue(
            safeVersion.safe.isModuleEnabled(address(saferSafes)),
            string.concat("Module should be enabled on ", safeVersion.version)
        );

        bytes32 guardSlot = bytes32(uint256(keccak256("guard_manager.guard.address")) - 1);
        bytes32 guardValue = vm.load(address(safeVersion.safe), guardSlot);
        address currentGuard = address(uint160(uint256(guardValue)));

        assertEq(currentGuard, address(saferSafes), string.concat("Guard should be set on ", safeVersion.version));
    }

    /// @notice Implementation: Full configuration including liveness module and timelock guard
    function _testFullConfiguration(SafeVersion memory safeVersion) external {
        // 1. Enable SaferSafes as a module
        bytes memory enableModuleData = abi.encodeCall(safeVersion.safe.enableModule, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, enableModuleData, Enum.Operation.Call);

        // 2. Set SaferSafes as guard
        bytes memory setGuardData = abi.encodeCall(safeVersion.safe.setGuard, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, setGuardData, Enum.Operation.Call);

        // 3. Configure the liveness module
        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: LIVENESS_RESPONSE_PERIOD,
            fallbackOwner: fallbackOwner
        });

        bytes memory configureModuleData = abi.encodeCall(saferSafes.configureLivenessModule, (moduleConfig));

        _executeSafeTransaction(safeVersion.safe, address(saferSafes), 0, configureModuleData, Enum.Operation.Call);

        // 4. Configure the timelock guard
        bytes memory configureGuardData = abi.encodeCall(saferSafes.configureTimelockGuard, (TIMELOCK_DELAY));

        _executeSafeTransaction(safeVersion.safe, address(saferSafes), 0, configureGuardData, Enum.Operation.Call);

        // Verify configurations
        (uint256 storedLivenessResponsePeriod, address storedFallbackOwner) =
            saferSafes.livenessSafeConfiguration(address(safeVersion.safe));

        assertEq(
            storedLivenessResponsePeriod,
            LIVENESS_RESPONSE_PERIOD,
            string.concat("Liveness response period should be set on ", safeVersion.version)
        );

        assertEq(
            storedFallbackOwner, fallbackOwner, string.concat("Fallback owner should be set on ", safeVersion.version)
        );

        assertEq(
            saferSafes.timelockConfiguration(safeVersion.safe),
            TIMELOCK_DELAY,
            string.concat("Timelock delay should be set on ", safeVersion.version)
        );
    }

    /// @notice Implementation: Test configuration validation
    /// @dev This should fail because liveness period (13 days) < 2 * timelock (14 days)
    function _testConfigurationValidation(SafeVersion memory safeVersion) external {
        // Enable module and guard first
        bytes memory enableModuleData = abi.encodeCall(safeVersion.safe.enableModule, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, enableModuleData, Enum.Operation.Call);

        bytes memory setGuardData = abi.encodeCall(safeVersion.safe.setGuard, (address(saferSafes)));
        _executeSafeTransaction(safeVersion.safe, address(safeVersion.safe), 0, setGuardData, Enum.Operation.Call);

        // Configure liveness module first with INVALID period (too short)
        uint256 invalidLivenessPeriod = 13 days; // This is < 2 * 7 days = 14 days

        LivenessModule2.ModuleConfig memory moduleConfig = LivenessModule2.ModuleConfig({
            livenessResponsePeriod: invalidLivenessPeriod,
            fallbackOwner: fallbackOwner
        });

        bytes memory configureModuleData = abi.encodeCall(saferSafes.configureLivenessModule, (moduleConfig));

        _executeSafeTransaction(safeVersion.safe, address(saferSafes), 0, configureModuleData, Enum.Operation.Call);

        // Now configure timelock guard - this SHOULD revert with validation error
        bytes memory configureGuardData = abi.encodeCall(saferSafes.configureTimelockGuard, (TIMELOCK_DELAY));

        // This should revert with SaferSafes_InsufficientLivenessResponsePeriod
        _executeSafeTransaction(safeVersion.safe, address(saferSafes), 0, configureGuardData, Enum.Operation.Call);

        // If we reach here in verbose mode, it means the validation didn't work
        // In strict mode, this would have already reverted
        revert(
            "SaferSafes_Uncategorized_Test: expected SaferSafes_InsufficientLivenessResponsePeriod but transaction succeeded"
        );
    }
}
