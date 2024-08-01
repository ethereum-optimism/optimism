// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { DeployConfig } from "scripts/deploy/DeployConfig.s.sol";

contract DeployConfigShim is Script {
    fallback() external payable virtual {
        // TODO get 4 byte selector of the config attribute we are trying to load
        // retrieve the data from the JSON
        // if not present, revert.
        // if present, then parse into typed value?
    }
}

contract Incremental is Script {

    DeployConfig public constant cfg =
    DeployConfig(address(uint160(uint256(keccak256(abi.encode("optimism.deployconfig"))))));

    function run() public {
        // preStateCachePath -> to load allocs from
        // postStateCachePath -> to write allocs to
        // call.Args -> patch DeployConfig contract values with this
        // call.Addrs -> patch Artifacts addresses with this
        // call.Prestate -> make it load this as initial state
        // call.Target -> magic identifier for deploy-script contract to call. Should be either the L1Deploy or L2Genesis script.
        // call.Sig -> call the same sig always, to handle the outer state loading/writing and patching of vars, but then call out to this method signature.

        vm.loadAllocs("todo.json");
        cfg.read("args.json"); // TODO

        // TODO insert Artifacts shim

        // TODO compute method signature bytes
        // TODO call contract with method signature

        vm.dumpState("todo.json");
        // TODO export deployments
        // TODO export labels
    }
}
