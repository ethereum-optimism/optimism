// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import {
    DeployOwnership,
    SafeConfig,
    SecurityCouncilConfig,
    DeputyGuardianModuleConfig,
    LivenessModuleConfig
} from "scripts/DeployOwnership.s.sol";
import { Test } from "forge-std/Test.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";
import { LivenessModule } from "src/Safe/LivenessModule.sol";
import { DeputyGuardianModule } from "src/Safe/DeputyGuardianModule.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

contract DeployOwnershipTest is Test, DeployOwnership {
    address internal constant SENTINEL_MODULES = address(0x1);
    // keccak256("guard_manager.guard.address")
    bytes32 internal constant GUARD_STORAGE_SLOT = 0x4a204f620c8c5ccdca3fd54d003badd85ba500436a431f0cbda4f558c93c34c8;

    function setUp() public override {
        super.setUp();
        run();
    }

    function _checkSafeConfig(SafeConfig memory _safeConfig, Safe _safe) internal view {
        assertEq(_safe.getThreshold(), _safeConfig.threshold);

        address[] memory safeOwners = _safe.getOwners();
        assertEq(_safeConfig.owners.length, safeOwners.length);
        for (uint256 i = 0; i < safeOwners.length; i++) {
            assertEq(safeOwners[i], _safeConfig.owners[i]);
        }
    }

    function test_exampleFoundationSafe() public {
        Safe foundationSafe = Safe(payable(mustGetAddress("FoundationSafe")));
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();

        _checkSafeConfig(exampleFoundationConfig, foundationSafe);
    }

    function test_exampleSecurityCouncilSafe() public {
        Safe securityCouncilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        SecurityCouncilConfig memory exampleSecurityCouncilConfig = _getExampleCouncilConfig();

        _checkSafeConfig(exampleSecurityCouncilConfig.safeConfig, securityCouncilSafe);

        // Guard Checks
        address livenessGuard = mustGetAddress("LivenessGuard");

        // The Safe's getGuard method is internal, so we read directly from storage
        // https://github.com/safe-global/safe-contracts/blob/v1.4.0/contracts/base/GuardManager.sol#L66-L72
        assertEq(vm.load(address(securityCouncilSafe), GUARD_STORAGE_SLOT), bytes32(uint256(uint160(livenessGuard))));

        // check that all the owners have a lastLive time in the Guard
        address[] memory owners = exampleSecurityCouncilConfig.safeConfig.owners;
        for (uint256 i = 0; i < owners.length; i++) {
            assertEq(LivenessGuard(livenessGuard).lastLive(owners[i]), block.timestamp);
        }

        // Module Checks
        address livenessModule = mustGetAddress("LivenessModule");
        address deputyGuardianModule = mustGetAddress("DeputyGuardianModule");
        (address[] memory modules, address nextModule) =
            ModuleManager(securityCouncilSafe).getModulesPaginated(SENTINEL_MODULES, 3);
        assertEq(modules.length, 2);
        assertEq(modules[0], livenessModule);
        assertEq(modules[1], deputyGuardianModule);
        assertEq(nextModule, SENTINEL_MODULES); // ensures there are no more modules in the list

        // DeputyGuardianModule checks
        DeputyGuardianModuleConfig memory dgmConfig = exampleSecurityCouncilConfig.deputyGuardianModuleConfig;
        assertEq(DeputyGuardianModule(deputyGuardianModule).deputyGuardian(), dgmConfig.deputyGuardian);
        assertEq(
            address(DeputyGuardianModule(deputyGuardianModule).superchainConfig()), address(dgmConfig.superchainConfig)
        );

        // LivenessModule checks
        LivenessModuleConfig memory lmConfig = exampleSecurityCouncilConfig.livenessModuleConfig;
        assertEq(address(LivenessModule(livenessModule).livenessGuard()), livenessGuard);
        assertEq(LivenessModule(livenessModule).livenessInterval(), lmConfig.livenessInterval);
        assertEq(LivenessModule(livenessModule).thresholdPercentage(), lmConfig.thresholdPercentage);
        assertEq(LivenessModule(livenessModule).minOwners(), lmConfig.minOwners);
    }
}
