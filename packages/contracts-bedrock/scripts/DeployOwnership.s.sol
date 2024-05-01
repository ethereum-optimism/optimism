// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import { Safe } from "safe-contracts/Safe.sol";
import { OwnerManager } from "safe-contracts/base/OwnerManager.sol";
import { GuardManager } from "safe-contracts/base/GuardManager.sol";
import { ModuleManager } from "safe-contracts/base/ModuleManager.sol";
import { SafeProxyFactory } from "safe-contracts/proxies/SafeProxyFactory.sol";
import { Enum as SafeOps } from "safe-contracts/common/Enum.sol";

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
    uint256 thresholdPercentage;
    uint256 minOwners;
    address fallbackOwner;
}

/// @title Deploy
/// @notice Script used to deploy and configure the Safe contracts which are used to manage the Superchain,
///         as the ProxyAdminOwner and other roles in the system.
contract DeployOwnership is Deploy {
    /// @notice Internal function containing the deploy logic.
    function _run() internal override {
        console.log("start of Ownership Deployment");
        deployAndConfigureFoundationSafe();
        deployAndConfigueSecurityCouncilSafe();
        console.log("deployed Security Council Safe!");
        console.log("Ownership contracts completed");
    }

    /// @notice Returns a SafeConfig with similar to that of the Foundation Safe on Mainnet.
    function _getExampleFoundationConfig() internal returns (SafeConfig memory safeConfig_) {
        address[] memory exampleFoundationOwners = new address[](7);
        for (uint256 i; i < exampleFoundationOwners.length; i++) {
            exampleFoundationOwners[i] = makeAddr(string(abi.encode(i)));
        }
        safeConfig_ = SafeConfig({ threshold: 5, owners: exampleFoundationOwners });
    }

    /// @notice Deploys a Safe with a configuration similar to that of the Foundation Safe on Mainnet.
    function deployAndConfigureFoundationSafe() public returns (address addr_) {
        address safe = deploySafe("FoundationSafe");
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
            _data: abi.encodeCall(OwnerManager.removeOwner, (address(0x1), msg.sender, exampleFoundationConfig.threshold))
        });
        addr_ = safe;
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
        address fallbackOwner = mustGetAddress("FoundationSafe");
        address guard = mustGetAddress("LivenessGuard");

        addr_ = address(
            new LivenessModule({
                _safe: councilSafe,
                _livenessGuard: LivenessGuard(guard),
                _livenessInterval: 1,
                _thresholdPercentage: 1,
                _minOwners: 1,
                _fallbackOwner: fallbackOwner
            })
        );

        save("LivenessModule", address(addr_));
        console.log("New LivenessModule deployed at %s", address(addr_));
    }

    /// @notice Deploy a Security Council with LivenessModule and LivenessGuard.
    function deployAndConfigueSecurityCouncilSafe() public returns (address addr_) {
        Safe safe = Safe(payable(deploySafe("SecurityCouncilSafe")));

        address guard = deployLivenessGuard();

        vm.startBroadcast();
        _callViaSafe({ _safe: safe, _target: address(safe), _data: abi.encodeCall(GuardManager.setGuard, (guard)) });
        console.log("LivenessGuard setup on SecurityCouncilSafe");

        address[] memory securityCouncilOwners = new address[](0);
        for (uint256 i = 0; i < securityCouncilOwners.length; i++) {
            _callViaSafe({
                _safe: safe,
                _target: address(safe),
                _data: abi.encodeCall(OwnerManager.addOwnerWithThreshold, (securityCouncilOwners[i], 1))
            });
        }

        // Now that the owners have been added, we can set the threshold to the desired value.
        uint256 newThreshold = 1;
        _callViaSafe({
            _safe: safe,
            _target: address(safe),
            _data: abi.encodeCall(OwnerManager.changeThreshold, (newThreshold))
        });

        // Now that the owners have been added and the threshold increased we can deploy the liveness module (otherwise
        // constructor checks will fail).
        address module = deployLivenessModule();

        // Unfortunately, a threshold of owners is required to actually enable the module, so we're unable to do that
        // here, and will settle for logging a warning below.
        addr_ = address(safe);
        console.log("New SecurityCouncilSafe deployed at %s", address(safe));
        console.log(
            string.concat(
                "\x1b[1;33mWARNING: The SecurityCouncilSafe is deployed with the LivenessGuard enabled.\n",
                "  The final setup will require a threshold of signers to\n",
                "    1. call enableModule() to enable the LivenessModule deployed at ",
                vm.toString(module),
                "\n",
                "    2. call `removeOwner() to remove the deployer with address ",
                vm.toString(msg.sender),
                " which is still an owner. The threshold should not be changed.\x1b[0m"
            )
        );
        vm.stopBroadcast();
    }
}
