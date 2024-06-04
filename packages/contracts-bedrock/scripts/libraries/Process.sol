// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Vm } from "forge-std/Vm.sol";

library Process {
    /// @notice Error for when an ffi command fails.
    error FfiFailed(string);

    /// @notice Foundry cheatcode VM.
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    function run(string[] memory cmd) internal returns (bytes memory stdout_) {
        Vm.FfiResult memory result = vm.tryFfi(cmd);
        if (result.exitCode != 0) {
            string memory command;
            for (uint256 i = 0; i < cmd.length; i++) {
                command = string.concat(command, cmd[i], " ");
            }
            revert FfiFailed(string.concat("Command: ", command, "\nError: ", string(result.stderr)));
        }
        stdout_ = result.stdout;
    }
}
