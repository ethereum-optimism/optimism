// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";

import { Deployer } from "scripts/deploy/Deployer.sol";

import { LivenessGuard } from "src/safe/LivenessGuard.sol";
import { LivenessModule } from "src/safe/LivenessModule.sol";
import { DeputyGuardianModule } from "src/safe/DeputyGuardianModule.sol";
import { ISuperchainConfig } from "src/L1/interfaces/ISuperchainConfig.sol";

import { Deploy } from "./Deploy.s.sol";

/// @notice Configuration for a Safe
struct SafeConfig {
    uint256 threshold;
    address[] owners;
}

/// @notice Configuration for the Liveness Module
struct LivenessModuleConfig {
    uint256 livenessInterval;
    uint256 thresholdPercentage;
    uint256 minOwners;
    address fallbackOwner;
}

/// @notice Configuration for the Security Council Safe.
struct SecurityCouncilConfig {
    SafeConfig safeConfig;
    LivenessModuleConfig livenessModuleConfig;
}

/// @notice Configuration for the Deputy Guardian Module
struct DeputyGuardianModuleConfig {
    address deputyGuardian;
    ISuperchainConfig superchainConfig;
}

/// @notice Configuration for the Guardian Safe.
struct GuardianConfig {
    SafeConfig safeConfig;
    DeputyGuardianModuleConfig deputyGuardianModuleConfig;
}

/// @title Deploy
/// @notice Script used to deploy and configure the Safe contracts which are used to manage the Superchain,
///         as the ProxyAdminOwner and other roles in the system. Note that this script is not executable in a
///         production environment as some steps depend on having a quorum of signers available. This script is meant to
///         be used as an example to guide the setup and configuration of the Safe contracts.
contract DeployOwnership is Deploy {
    /// @notice Internal function containing the deploy logic.
    function _run() internal override {
        console.log("start of Ownership Deployment");
        // The SuperchainConfig is needed as a constructor argument to the Deputy Guardian Module
        deploySuperchainConfig();

        deployFoundationOperationsSafe();
        deployFoundationUpgradeSafe();
        deploySecurityCouncilSafe();
        deployGuardianSafe();
        configureGuardianSafe();
        configureSecurityCouncilSafe();

        console.log("Ownership contracts completed");
    }

    /// @notice Returns a SafeConfig similar to that of the Foundation Safe on Mainnet.
    function _getExampleFoundationConfig() internal returns (SafeConfig memory safeConfig_) {
        address[] memory exampleFoundationOwners = new address[](7);
        for (uint256 i; i < exampleFoundationOwners.length; i++) {
            exampleFoundationOwners[i] = makeAddr(string.concat("fnd-", vm.toString(i)));
        }
        safeConfig_ = SafeConfig({ threshold: 5, owners: exampleFoundationOwners });
    }

    /// @notice Returns a GuardianConfig similar to that of the Guardian Safe on Mainnet.
    function _getExampleGuardianConfig() internal view returns (GuardianConfig memory guardianConfig_) {
        address[] memory exampleGuardianOwners = new address[](1);
        exampleGuardianOwners[0] = mustGetAddress("SecurityCouncilSafe");
        guardianConfig_ = GuardianConfig({
            safeConfig: SafeConfig({ threshold: 1, owners: exampleGuardianOwners }),
            deputyGuardianModuleConfig: DeputyGuardianModuleConfig({
                deputyGuardian: mustGetAddress("FoundationOperationsSafe"),
                superchainConfig: ISuperchainConfig(mustGetAddress("SuperchainConfig"))
            })
        });
    }

    /// @notice Returns a SafeConfig similar to that of the Security Council Safe on Mainnet.
    function _getExampleCouncilConfig() internal returns (SecurityCouncilConfig memory councilConfig_) {
        address[] memory exampleCouncilOwners = new address[](13);
        for (uint256 i; i < exampleCouncilOwners.length; i++) {
            exampleCouncilOwners[i] = makeAddr(string.concat("sc-", vm.toString(i)));
        }
        SafeConfig memory safeConfig = SafeConfig({ threshold: 10, owners: exampleCouncilOwners });
        councilConfig_ = SecurityCouncilConfig({
            safeConfig: safeConfig,
            livenessModuleConfig: LivenessModuleConfig({
                livenessInterval: 14 weeks,
                thresholdPercentage: 75,
                minOwners: 8,
                fallbackOwner: mustGetAddress("FoundationUpgradeSafe")
            })
        });
    }

    /// @notice Deploys a Safe with a configuration similar to that of the Foundation Safe on Mainnet.
    function deployFoundationOperationsSafe() public broadcast returns (address addr_) {
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();
        addr_ = deploySafe({
            _name: "FoundationOperationsSafe",
            _owners: exampleFoundationConfig.owners,
            _threshold: exampleFoundationConfig.threshold,
            _keepDeployer: false
        });
    }

    /// @notice Deploys a Safe with a configuration similar to that of the Foundation Safe on Mainnet.
    function deployFoundationUpgradeSafe() public broadcast returns (address addr_) {
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();
        addr_ = deploySafe({
            _name: "FoundationUpgradeSafe",
            _owners: exampleFoundationConfig.owners,
            _threshold: exampleFoundationConfig.threshold,
            _keepDeployer: false
        });
    }

    /// @notice Deploy a LivenessGuard for use on the Security Council Safe.
    ///         Note this function does not have the broadcast modifier.
    function deployLivenessGuard() public returns (address addr_) {
        Safe councilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        addr_ = address(new LivenessGuard(councilSafe));

        save("LivenessGuard", address(addr_));
        console.log("New LivenessGuard deployed at %s", address(addr_));
    }

    /// @notice Deploy a LivenessModule for use on the Security Council Safe
    ///         Note this function does not have the broadcast modifier.
    function deployLivenessModule() public returns (address addr_) {
        Safe councilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        address guard = mustGetAddress("LivenessGuard");
        LivenessModuleConfig memory livenessModuleConfig = _getExampleCouncilConfig().livenessModuleConfig;

        addr_ = address(
            new LivenessModule({
                _safe: councilSafe,
                _livenessGuard: LivenessGuard(guard),
                _livenessInterval: livenessModuleConfig.livenessInterval,
                _thresholdPercentage: livenessModuleConfig.thresholdPercentage,
                _minOwners: livenessModuleConfig.minOwners,
                _fallbackOwner: livenessModuleConfig.fallbackOwner
            })
        );

        save("LivenessModule", address(addr_));
        console.log("New LivenessModule deployed at %s", address(addr_));
    }

    /// @notice Deploy a DeputyGuardianModule for use on the Security Council Safe.
    ///         Note this function does not have the broadcast modifier.
    function deployDeputyGuardianModule() public returns (address addr_) {
        Safe guardianSafe = Safe(payable(mustGetAddress("GuardianSafe")));
        DeputyGuardianModuleConfig memory deputyGuardianModuleConfig =
            _getExampleGuardianConfig().deputyGuardianModuleConfig;
        addr_ = address(
            new DeputyGuardianModule({
                _safe: guardianSafe,
                _superchainConfig: deputyGuardianModuleConfig.superchainConfig,
                _deputyGuardian: deputyGuardianModuleConfig.deputyGuardian
            })
        );

        save("DeputyGuardianModule", addr_);
        console.log("New DeputyGuardianModule deployed at %s", addr_);
    }

    /// @notice Deploy a Security Council Safe.
    function deploySecurityCouncilSafe() public broadcast returns (address addr_) {
        // Deploy the safe with the extra deployer key, and keep the threshold at 1 to allow for further setup.
        SecurityCouncilConfig memory exampleCouncilConfig = _getExampleCouncilConfig();
        addr_ = payable(
            deploySafe({
                _name: "SecurityCouncilSafe",
                _owners: exampleCouncilConfig.safeConfig.owners,
                _threshold: 1,
                _keepDeployer: true
            })
        );
    }

    /// @notice Deploy Guardian Safe.
    function deployGuardianSafe() public broadcast returns (address addr_) {
        // Config is hardcoded here as the Guardian Safe's configuration is inflexible.
        address[] memory owners = new address[](1);
        owners[0] = mustGetAddress("SecurityCouncilSafe");
        addr_ = deploySafe({ _name: "GuardianSafe", _owners: owners, _threshold: 1, _keepDeployer: true });

        console.log("Deployed and configured the Guardian Safe!");
    }

    /// @notice Configure the Guardian Safe with the DeputyGuardianModule.
    function configureGuardianSafe() public broadcast returns (address addr_) {
        addr_ = mustGetAddress("GuardianSafe");
        address deputyGuardianModule = deployDeputyGuardianModule();
        _callViaSafe({
            _safe: Safe(payable(addr_)),
            _target: addr_,
            _data: abi.encodeCall(ModuleManager.enableModule, (deputyGuardianModule))
        });

        // Finalize configuration by removing the additional deployer key.
        removeDeployerFromSafe({ _name: "GuardianSafe", _newThreshold: 1 });
        console.log("DeputyGuardianModule enabled on GuardianSafe");
    }

    /// @notice Configure the Security Council Safe with the LivenessModule and LivenessGuard.
    function configureSecurityCouncilSafe() public broadcast returns (address addr_) {
        // Deploy and add the Deputy Guardian Module.
        SecurityCouncilConfig memory exampleCouncilConfig = _getExampleCouncilConfig();
        Safe safe = Safe(mustGetAddress("SecurityCouncilSafe"));

        // Deploy and add the Liveness Guard.
        address guard = deployLivenessGuard();
        _callViaSafe({ _safe: safe, _target: address(safe), _data: abi.encodeCall(GuardManager.setGuard, (guard)) });
        console.log("LivenessGuard setup on SecurityCouncilSafe");

        // Deploy and add the Liveness Module.
        address livenessModule = deployLivenessModule();
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(ModuleManager.enableModule, (livenessModule))
        });

        // Finalize configuration by removing the additional deployer key.
        removeDeployerFromSafe({ _name: "SecurityCouncilSafe", _newThreshold: exampleCouncilConfig.safeConfig.threshold });

        address[] memory owners = safe.getOwners();
        require(
            safe.getThreshold() == LivenessModule(livenessModule).getRequiredThreshold(owners.length),
            "Safe threshold must be equal to the LivenessModule's required threshold"
        );

        addr_ = address(safe);
        console.log("Deployed and configured the Security Council Safe!");
    }
}
