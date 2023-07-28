// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Script } from "forge-std/Script.sol";
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

            // Serialize the source hash in JSON.
            string memory j = vm.serializeBytes32(out, _files[i], keccak256(abi.encodePacked(fileContents)));

            // If this is the last file, set the output.
            if (i == _files.length - 1) {
                out = j;
            }
        }

        // Write the semver lockfile.
        vm.writeJson(out, "semver-lock.json");
        console.logString("Wrote semver lock file to \"semver-lock.json\".");
    }
}
