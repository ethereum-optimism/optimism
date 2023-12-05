// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { VmSafe } from "forge-std/Vm.sol";
import { Script } from "forge-std/Script.sol";

import { console2 as console } from "forge-std/console2.sol";
import { stdJson } from "forge-std/StdJson.sol";

import {Counter} from "src/L1/Counter.sol";
import { LibStateDiff } from "scripts/libraries/LibStateDiff.sol";

contract MakeStateDiff is Script {

    function name() public pure returns (string memory name_) {
        name_ = "TestDiff";
    }

    /// @notice Modifier that wraps a function in broadcasting.
    modifier broadcast() {
        vm.startBroadcast(msg.sender);
        _;
        vm.stopBroadcast();
    }

    /// @notice Modifier that wraps a function with statediff recording.
    ///         The returned AccountAccess[] array is then written to
    ///         the `snapshots/state-diff/<name>.json` output file.
    modifier stateDiff() {
        vm.startStateDiffRecording();
        _;
        VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();
        console.log("ACCESSES CREATE COUNTER", accesses[0].account);
        console.log("ACCESSES CREATE COUNTER2", accesses[1].account);
        console.log("Writing %d state diff account accesses to snapshots/state-diff/%s.json", accesses.length, name());
        string memory json = LibStateDiff.encodeAccountAccesses(accesses);
        string memory statediffPath = string.concat(vm.projectRoot(), "/snapshots/state-diff/", name(), ".json");
        vm.writeJson({ json: json, path: statediffPath });
    }

    function testStateDiff() public stateDiff /* broadcast */ {
        Counter counter = new Counter();
        console.log("COUNTER", address(counter));
        Counter counter2 = new Counter();
        console.log("COUNTER2", address(counter2));
        counter.setNumber(3);
        counter.setNumber(42);
        counter2.setNumber(777);
        /* sync(); */
    }
}
