// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console2 as console } from "forge-std/console2.sol";

contract SemverLock is Script {
    function run() public {
        // First, find all contracts with a Semver inheritance.
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = "grep -rl '@custom:semver' src | jq -Rs 'split(\"\\n\") | map(select(length > 0))'";
        string memory rawFiles = string(vm.ffi(commands));

        string[] memory files = vm.parseJsonStringArray(rawFiles, "");
        writeSemverLock(files);
    }

    /// @dev Writes a Semver lockfile
    function writeSemverLock(string[] memory _files) internal {
        string memory out;
        for (uint256 i; i < _files.length; i++) {
            // Use FFI to read the file to remove the need for FS permissions in the foundry.toml.
            string[] memory commands = new string[](2);
            commands[0] = "cat";
            commands[1] = _files[i];
            string memory fileContents = string(vm.ffi(commands));

            // Grab the contract name
            commands = new string[](3);
            commands[0] = "bash";
            commands[1] = "-c";
            commands[2] = string.concat("echo \"", _files[i], "\"| sed -E \'s|src/.*/(.+)\\.sol|\\1|\'");
            string memory contractName = string(vm.ffi(commands));

            commands[2] = "forge config --json | jq -r .out";
            string memory artifactsDir = string(vm.ffi(commands));

            // Handle the case where there are multiple artifacts for a contract. This happens
            // when the same contract is compiled with multiple compiler versions.
            string memory contractArtifactDir = string.concat(artifactsDir, "/", contractName, ".sol");
            commands[2] = string.concat(
                "ls -1 --color=never ", contractArtifactDir, " | jq -R -s -c 'split(\"\n\") | map(select(length > 0))'"
            );
            string memory artifactFiles = string(vm.ffi(commands));

            string[] memory files = stdJson.readStringArray(artifactFiles, "");
            require(files.length > 0, string.concat("No artifacts found for ", contractName));
            string memory fileName = files[0];

            // Parse the artifact to get the contract's initcode hash.
            bytes memory initCode = vm.getCode(string.concat(artifactsDir, "/", contractName, ".sol/", fileName));

            // Serialize the initcode hash + sourcecode hash in JSON.
            vm.serializeBytes32(_files[i], "initCodeHash", keccak256(initCode));
            string memory obj = vm.serializeBytes32(_files[i], "sourceCodeHash", keccak256(bytes(fileContents)));

            // Serialize the map from the file name -> initcode hash + sourcecode hash in JSON.
            out = vm.serializeString("", _files[i], obj);
        }

        // Write the semver lockfile.
        vm.writeJson(out, "semver-lock.json");
        console.logString("Wrote semver lock file to \"semver-lock.json\".");
    }
}
