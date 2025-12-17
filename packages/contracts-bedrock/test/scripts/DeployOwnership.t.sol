// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    DeployOwnership,
    SafeConfig,
    SecurityCouncilConfig,
    LivenessModuleConfig
} from "scripts/deploy/DeployOwnership.s.sol";
import { Test } from "forge-std/Test.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";

import { LivenessModule2 } from "src/safe/LivenessModule2.sol";

contract DeployOwnershipTest is Test, DeployOwnership {
    address internal constant SENTINEL_MODULES = address(0x1);

    function setUp() public override {
        super.setUp();
        run();
    }

    /// @dev Helper function to make assertions on basic Safe config properties.
    function _checkSafeConfig(SafeConfig memory _safeConfig, Safe _safe) internal view {
        assertEq(_safe.getThreshold(), _safeConfig.threshold);

        address[] memory safeOwners = _safe.getOwners();
        assertEq(_safeConfig.owners.length, safeOwners.length);
        assertFalse(_safe.isOwner(msg.sender));
        for (uint256 i = 0; i < safeOwners.length; i++) {
            assertEq(safeOwners[i], _safeConfig.owners[i]);
        }
    }

    /// @dev Test the example Foundation Safe configurations, against the expected configuration, and
    ///     check that they both have the same configuration.
    function test_exampleFoundationSafes_configuration_succeeds() public {
        Safe upgradeSafe = Safe(payable(artifacts.mustGetAddress("FoundationUpgradeSafe")));
        Safe operationsSafe = Safe(payable(artifacts.mustGetAddress("FoundationOperationsSafe")));
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();

        // Ensure the safes both match the example configuration
        _checkSafeConfig(exampleFoundationConfig, upgradeSafe);
        _checkSafeConfig(exampleFoundationConfig, operationsSafe);

        // Sanity check to ensure the safes match each other's configuration
        assertEq(upgradeSafe.getThreshold(), operationsSafe.getThreshold());
        assertEq(upgradeSafe.getOwners().length, operationsSafe.getOwners().length);
    }

    /// @dev Test the example Security Council Safe configuration.
    function test_exampleSecurityCouncilSafe_configuration_succeeds() public {
        Safe securityCouncilSafe = Safe(payable(artifacts.mustGetAddress("SecurityCouncilSafe")));
        SecurityCouncilConfig memory exampleSecurityCouncilConfig = _getExampleCouncilConfig();

        _checkSafeConfig(exampleSecurityCouncilConfig.safeConfig, securityCouncilSafe);

        // Module Checks
        address livenessModule = artifacts.mustGetAddress("LivenessModule2");
        (address[] memory modules, address nextModule) =
            ModuleManager(securityCouncilSafe).getModulesPaginated(SENTINEL_MODULES, 2);
        assertEq(modules.length, 1);
        assertEq(modules[0], livenessModule);
        assertEq(nextModule, SENTINEL_MODULES); // ensures there are no more modules in the list

        // LivenessModule2 checks
        LivenessModuleConfig memory lmConfig = exampleSecurityCouncilConfig.livenessModuleConfig;
        LivenessModule2.ModuleConfig memory moduleConfig =
            LivenessModule2(livenessModule).livenessSafeConfiguration(Safe(payable(securityCouncilSafe)));
        assertEq(moduleConfig.livenessResponsePeriod, lmConfig.livenessInterval);
        assertEq(moduleConfig.fallbackOwner, lmConfig.fallbackOwner);

        // Verify no active challenge exists initially
        assertEq(LivenessModule2(livenessModule).getChallengePeriodEnd(Safe(payable(securityCouncilSafe))), 0);
    }
}
