// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";

import { Deployer } from "scripts/Deployer.sol";

import { LivenessGuard } from "src/Safe/LivenessGuard.sol";
import { LivenessModule } from "src/Safe/LivenessModule.sol";

import { Deploy } from "./Deploy.s.sol";

struct SafeConfig {
    uint256 threshold;
    address[] owners;
}

struct SecurityCouncilConfig {
    SafeConfig safeConfig;
    uint256 livenessInterval;
    uint256 thresholdPercentage;
    uint256 minOwners;
    address fallbackOwner;
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
            livenessInterval: 24 weeks,
            thresholdPercentage: 75,
            minOwners: 8,
            fallbackOwner: mustGetAddress("FoundationSafe")
        });
    }

    /// @notice Deploys a Safe with a configuration similar to that of the Foundation Safe on Mainnet.
    function deployAndConfigureFoundationSafe() public returns (address addr_) {
        address safe = deploySafe("FoundationSafe"); // This function has a `broadcast` modifier

        vm.startBroadcast(msg.sender);
        SafeConfig memory exampleFoundationConfig = _getExampleFoundationConfig();
        for (uint256 i; i < exampleFoundationConfig.owners.length; i++) {
            _callViaSafe({
                _safe: Safe(payable(safe)),
                _target: safe,
                _data: abi.encodeCall(OwnerManager.addOwnerWithThreshold, (exampleFoundationConfig.owners[i], 1))
            });
        }
        _callViaSafe({
            _safe: Safe(payable(safe)),
            _target: safe,
            _data: abi.encodeCall(
                OwnerManager.removeOwner, (exampleFoundationConfig.owners[0], msg.sender, exampleFoundationConfig.threshold)
            )
        });
        addr_ = safe;
        vm.stopBroadcast();
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
        SecurityCouncilConfig memory councilConfig = _getExampleCouncilConfig();

        addr_ = address(
            new LivenessModule({
                _safe: councilSafe,
                _livenessGuard: LivenessGuard(guard),
                _livenessInterval: councilConfig.livenessInterval,
                _thresholdPercentage: councilConfig.thresholdPercentage,
                _minOwners: councilConfig.minOwners,
                _fallbackOwner: councilConfig.fallbackOwner
            })
        );

        save("LivenessModule", address(addr_));
        console.log("New LivenessModule deployed at %s", address(addr_));
    }

    /// @notice Deploy a Security Council with LivenessModule and LivenessGuard.
    function deployAndConfigureSecurityCouncilSafe() public returns (address addr_) {
        Safe safe = Safe(payable(deploySafe("SecurityCouncilSafe")));

        address guard = deployLivenessGuard();

        vm.startBroadcast();
        _callViaSafe({ _safe: safe, _target: address(safe), _data: abi.encodeCall(GuardManager.setGuard, (guard)) });
        console.log("LivenessGuard setup on SecurityCouncilSafe");

        SecurityCouncilConfig memory exampleCouncilConfig = _getExampleCouncilConfig();
        // Add the owners, keeping the threshold at 1 for now.
        for (uint256 i = 0; i < exampleCouncilConfig.safeConfig.owners.length; i++) {
            _callViaSafe({
                _safe: safe,
                _target: address(safe),
                _data: abi.encodeCall(OwnerManager.addOwnerWithThreshold, (exampleCouncilConfig.safeConfig.owners[i], 1))
            });
        }
        // Remove the deployer address which was used to setup the Security Council Safe thus far
        // this call is also used to update the threshold.
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(
                OwnerManager.removeOwner,
                (exampleCouncilConfig.safeConfig.owners[0], msg.sender, exampleCouncilConfig.safeConfig.threshold)
            )
        });

        address livenessModule = deployLivenessModule();
        vm.stopBroadcast();

        // Since we don't have private keys for the safe owners, we instead use 'startBroadcast' to do something
        // similar to pranking as the safe. This simulates a quorum of signers executing a transation from the safe to
        // call it's own 'enableModule' method.
        vm.startBroadcast(address(safe));
        safe.enableModule(livenessModule);
        addr_ = address(safe);
        console.log("Deployed and configured the Security Council Safe!");
    }
}
