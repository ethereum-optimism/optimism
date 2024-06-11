// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Vm } from "forge-std/Vm.sol";
import { Process } from "scripts/libraries/Process.sol";

/// @notice The executables used in ffi commands. These are set here
///         to have a single source of truth in case absolute paths
///         need to be used.
library Executables {
    /// @notice Foundry cheatcode VM.
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));
    string internal constant bash = "bash";
    string internal constant jq = "jq";
    string internal constant forge = "forge";
    string internal constant echo = "echo";
    string internal constant sed = "sed";
    string internal constant find = "find";
    string internal constant ls = "ls";
    string internal constant git = "git";

    /// @notice Returns the commit hash of HEAD.
    function gitCommitHash() internal returns (string memory) {
        string[] memory commands = new string[](3);
        commands[0] = bash;
        commands[1] = "-c";
        commands[2] = "cast abi-encode 'f(string)' $(git rev-parse HEAD)";
        return abi.decode(Process.run(commands), (string));
    }
}
