// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { VmSafe } from "forge-std/Vm.sol";
// import { Script } from "forge-std/Script.sol";

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

/// @title Deploy
/// @notice Script used to deploy and configure the Safe contracts which are used to manage the Superchain,
///         as the ProxyAdminOwner and other roles in the system.
contract DeployOwnership is Deploy {
    /// @notice Internal function containing the deploy logic.
    function _run() internal override {
        console.log("start of Ownership Deployment");
        deployFoundationSafe();
        console.log("deployed System Owner Safe!");
        deploySecurityCouncilSafe();
        console.log("deployed Security Council Safe!");
        console.log("Ownership contracts completed");
    }

    /// @notice Deploy the SystemOwnerSafe
    function deployFoundationSafe() public broadcast returns (address addr_) {
        console.log("Deploying System Owner Safe");
        (SafeProxyFactory safeProxyFactory, Safe safeSingleton) = _getSafeFactory();

        address[] memory signers = new address[](1);
        signers[0] = msg.sender;

        bytes memory initData =
            abi.encodeCall(Safe.setup, (signers, 1, address(0), hex"", address(0), address(0), 0, payable(address(0))));
        address safe = address(safeProxyFactory.createProxyWithNonce(address(safeSingleton), initData, block.timestamp));

        save("SystemOwnerSafe", address(safe));
        console.log("New SystemOwnerSafe deployed at %s", address(safe));
        addr_ = safe;
    }

    /// @notice Deploy a LivenessGuard for use on the Security Council Safe.
    ///         Note this function does not have the broadcast modifier.
    function deployLivenessGuard() public returns (address guard_) {
        Safe councilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        guard_ = address(new LivenessGuard(councilSafe));

        save("LivenessGuard", address(guard_));
        console.log("New LivenessGuard deployed at %s", address(guard_));
    }

    /// @notice Deploy a LivenessModule for use on the Security Council Safe
    ///         Note this function does not have the broadcast modifier.
    function deployLivenessModule() public returns (address module_) {
        Safe councilSafe = Safe(payable(mustGetAddress("SecurityCouncilSafe")));
        address fallbackOwner = mustGetAddress("SystemOwnerSafe");
        address guard = mustGetAddress("LivenessGuard");

        module_ = address(
            new LivenessModule({
                _safe: councilSafe,
                _livenessGuard: LivenessGuard(guard),
                _livenessInterval: cfg.livenessModuleInterval(),
                _thresholdPercentage: cfg.livenessModuleThresholdPercentage(),
                _minOwners: cfg.livenessModuleMinOwners(),
                _fallbackOwner: fallbackOwner
            })
        );

        save("LivenessModule", address(module_));
        console.log("New LivenessModule deployed at %s", address(module_));
    }

    /// @notice Deploy a Security Council with LivenessModule and LivenessGuard.
    function deploySecurityCouncilSafe() public broadcast returns (address addr_) {
        console.log("Deploying Security Council Safe");
        (SafeProxyFactory safeProxyFactory, Safe safeSingleton) = _getSafeFactory();

        address[] memory initialSigners = new address[](1);
        initialSigners[0] = msg.sender;

        bytes memory initData = abi.encodeCall(
            Safe.setup, (initialSigners, 1, address(0), hex"", address(0), address(0), 0, payable(address(0)))
        );
        Safe safe = Safe(
            payable(
                address(safeProxyFactory.createProxyWithNonce(address(safeSingleton), initData, block.timestamp + 1))
            )
        );

        save("SecurityCouncilSafe", address(safe));
        console.log("New SecurityCouncilSafe deployed at %s", address(safe));

        address guard = deployLivenessGuard();
        _callViaSafe({ _safe: safe, _target: address(safe), _data: abi.encodeCall(GuardManager.setGuard, (guard)) });
        console.log("LivenessGuard setup on SecurityCouncilSafe");

        address[] memory securityCouncilOwners = cfg.securityCouncilOwners();
        for (uint256 i = 0; i < securityCouncilOwners.length; i++) {
            _callViaSafe({
                _safe: safe,
                _target: address(safe),
                _data: abi.encodeCall(OwnerManager.addOwnerWithThreshold, (securityCouncilOwners[i], 1))
            });
        }

        // Now that the owners have been added, we can set the threshold to the desired value.
        uint256 newThreshold = cfg.securityCouncilThreshold();
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
    }
}
