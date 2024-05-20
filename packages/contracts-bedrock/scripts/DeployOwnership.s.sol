// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { GnosisSafe as Safe } from "safe-contracts/GnosisSafe.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";

import { Deployer } from "scripts/Deployer.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";
import { LivenessModule } from "src/Safe/LivenessModule.sol";
import { DeputyGuardianModule } from "src/Safe/DeputyGuardianModule.sol";
import { SuperchainConfig } from "src/L1/SuperchainConfig.sol";

import { Deploy } from "./Deploy.s.sol";

/// @notice Configuration for a Safe
struct SafeConfig {
    uint256 threshold;
    address[] owners;
}

struct LivenessModuleConfig {
    uint256 livenessInterval;
    uint256 thresholdPercentage;
    uint256 minOwners;
    address fallbackOwner;
}

struct DeputyGuardianModuleConfig {
    address deputyGuardian;
    SuperchainConfig superchainConfig;
}
/// @notice Configuration for the Security Council Safe.

struct SecurityCouncilConfig {
    SafeConfig safeConfig;
    LivenessModuleConfig livenessModuleConfig;
    DeputyGuardianModuleConfig deputyGuardianModuleConfig;
}

// The sentinel address is used to mark the start and end of the linked list of owners in the Safe.
address constant SENTINEL_OWNERS = address(0x1);

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
        deployAndConfigureFoundationSafe();
        deployAndConfigureSecurityCouncilSafe();

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
                livenessInterval: 24 weeks,
                thresholdPercentage: 75,
                minOwners: 8,
                fallbackOwner: mustGetAddress("FoundationSafe")
            }),
            deputyGuardianModuleConfig: DeputyGuardianModuleConfig({
                deputyGuardian: mustGetAddress("FoundationSafe"),
                superchainConfig: SuperchainConfig(mustGetAddress("SuperchainConfig"))
            })
        });
    }

    /// @notice Deploys a Safe with a configuration similar to that of the Foundation Safe on Mainnet.
    function deployAndConfigureFoundationSafe() public returns (address addr_) {
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();
        addr_ = deploySafe({
            _name: "FoundationSafe",
            _owners: exampleFoundationConfig.owners,
            _threshold: exampleFoundationConfig.threshold,
            _keepDeployer: false
        });
        console.log("Deployed and configured the Foundation Safe!");
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
        SecurityCouncilConfig memory councilConfig = _getExampleCouncilConfig();
        Safe councilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        DeputyGuardianModuleConfig memory deputyGuardianModuleConfig = councilConfig.deputyGuardianModuleConfig;
        addr_ = address(
            new DeputyGuardianModule({
                _safe: councilSafe,
                _superchainConfig: deputyGuardianModuleConfig.superchainConfig,
                _deputyGuardian: deputyGuardianModuleConfig.deputyGuardian
            })
        );

        save("DeputyGuardianModule", addr_);
        console.log("New DeputyGuardianModule deployed at %s", addr_);
    }

    /// @notice Deploy a Security Council with LivenessModule and LivenessGuard.
    function deployAndConfigureSecurityCouncilSafe() public returns (address addr_) {
        // Deploy the safe with the extra deployer key, and keep the threshold at 1 to allow for further setup.
        SecurityCouncilConfig memory exampleCouncilConfig = _getExampleCouncilConfig();
        Safe safe = Safe(
            payable(
                deploySafe({
                    _name: "SecurityCouncilSafe",
                    _owners: exampleCouncilConfig.safeConfig.owners,
                    _threshold: 1,
                    _keepDeployer: true
                })
            )
        );

        vm.startBroadcast(msg.sender);
        // Deploy and add the Deputy Guardian Module.
        address deputyGuardianModule = deployDeputyGuardianModule();
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(ModuleManager.enableModule, (deputyGuardianModule))
        });
        console.log("DeputyGuardianModule enabled on SecurityCouncilSafe");

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

        // Remove the deployer address (msg.sender) which was used to setup the Security Council Safe thus far
        // this call is also used to update the threshold.
        // Because deploySafe() always adds msg.sender first (if keepDeployer is true), we know that the previousOwner
        // will be SENTINEL_OWNERS.
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(
                OwnerManager.removeOwner, (SENTINEL_OWNERS, msg.sender, exampleCouncilConfig.safeConfig.threshold)
            )
        });
        address[] memory owners = safe.getOwners();
        require(
            safe.getThreshold() == LivenessModule(livenessModule).getRequiredThreshold(owners.length),
            "Safe threshold must be equal to the LivenessModule's required threshold"
        );
        addr_ = address(safe);
        console.log("Deployed and configured the Security Council Safe!");
    }
}
