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

    /// @notice Modifier that wraps a function with statediff recording.
    ///         The returned AccountAccess[] array is then written to
    ///         the `snapshots/state-diff/<name>.json` output file.
    modifier stateDiff() {
        vm.startStateDiffRecording();
        _;
        VmSafe.AccountAccess[] memory accesses = vm.stopAndReturnStateDiff();
        console.log("Writing %d state diff account accesses to snapshots/state-diff/%s.json", accesses.length, name());
        string memory json = LibStateDiff.encodeAccountAccesses(accesses);
        string memory statediffPath = string.concat(vm.projectRoot(), "/snapshots/state-diff/", name(), ".json");
        vm.writeJson({ json: json, path: statediffPath });
    }

    function testStateDiff() public stateDiff {
        Counter counter = new Counter();
        counter.setNumber(3);
    }
}
