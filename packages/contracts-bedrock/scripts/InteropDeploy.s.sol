// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { Deploy } from "scripts/Deploy.s.sol";
import { Config } from "scripts/Config.sol";
import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { Executables } from "scripts/Executables.sol";
import { console } from "forge-std/console.sol";

/// @title InteropDeployer
/// @author jinmel
/// @notice A contract that can make deploying and interacting with deployments easy.
contract InteropDeploy {

    function run() public {
        uint256 id = vm.snapshot();
        vm.setEnv("DEPLOYMENT_OUTFILE", vm.envOr("DEPLOYMENT_OUTFILE", ""));
        vm.setEnv("DEPLOY_CONFIG_PATH", vm.envOr("DEPLOY_CONFIG_PATH", ""));

        Deploy.setUp();
        Deploy.runWithStateDump();

        vm.revert(id);

        vm.setEnv("DEPLOYMENT_OUTFILE", vm.envOr("DEPLOYMENT_OUTFILE_INTEROP", ""));
        vm.setEnv("DEPLOY_CONFIG_PATH", vm.envOr("DEPLOY_CONFIG_PATH_INTEROP", ""));

        Deploy.setUp();
        Deploy.runWithStateDump();

        vm.loadAllocs("path-to-allocs.json");
        vm.loadAllocs("path-to-allocs.json");
        vm.dumpState("final-state.json");
    }
}
