// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { Artifacts } from "scripts/Artifacts.s.sol";
import { Deploy } from "scripts/deploy/Deploy.s.sol";
import { Config } from "scripts/Config.sol";
import { console } from "forge-std/console.sol";

/// @title InteropDeployer
/// @author jinmel
/// @notice A contract that can make deploying and interacting with deployments easy.
contract InteropDeploy is Deploy {
    // @notice Deploys all of the L1 contracts necessary for two rollups configured by five env vars.
    //         The first rollup(default) is configured by DEPLOYMENT_OUTFILE and DEPLOY_CONFIG_PATH.
    //         The second rollup(interop) is configured by INTEROP_DEPLOYMENT_OUTFILE and INTEROP_DEPLOY_CONFIG_PATH.
    //         The final state-dump is written to STATE_DUMP_PATH
    function runWithStateDump() public override {
        uint256 id = vm.snapshot();

        vm.setEnv("DEPLOYMENT_OUTFILE", vm.envString("DEPLOYMENT_OUTFILE"));
        vm.setEnv("DEPLOY_CONFIG_PATH", vm.envString("DEPLOY_CONFIG_PATH"));

        console.log("Deploying default rollup");
        Artifacts.setUp();
        vm.chainId(cfg.l1ChainID());
        Deploy.run();
        vm.dumpState(Config.stateDumpPath("-default"));
        vm.revertTo(id);

        console.log("Deploying interop rollup");
        vm.setEnv("DEPLOYMENT_OUTFILE", vm.envString("INTEROP_DEPLOYMENT_OUTFILE"));
        vm.setEnv("DEPLOY_CONFIG_PATH", vm.envString("INTEROP_DEPLOY_CONFIG_PATH"));
        Artifacts.setUp();
        vm.chainId(cfg.l1ChainID());
        Deploy.run();
        vm.dumpState(Config.stateDumpPath("-interop"));
        vm.revertTo(id);

        vm.loadAllocs(Config.stateDumpPath("-default"));
        vm.loadAllocs(Config.stateDumpPath("-interop"));
        vm.dumpState(Config.stateDumpPath(""));
    }
}
