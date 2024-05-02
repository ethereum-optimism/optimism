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

/// @notice Configuration for a Safe
struct SafeConfig {
    uint256 threshold;
    address[] owners;
}

/// @notice Configuration for the Security Council Safe.
struct SecurityCouncilConfig {
    SafeConfig safeConfig;
    uint256 livenessInterval;
    uint256 thresholdPercentage;
    uint256 minOwners;
    address fallbackOwner;
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

        vm.startBroadcast();
        address guard = deployLivenessGuard();
        _callViaSafe({ _safe: safe, _target: address(safe), _data: abi.encodeCall(GuardManager.setGuard, (guard)) });
        console.log("LivenessGuard setup on SecurityCouncilSafe");

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
