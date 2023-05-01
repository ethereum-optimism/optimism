// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Vm } from "forge-std/Vm.sol";
import { stdJson } from "forge-std/StdJson.sol";

contract JSONConfig {
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    string private _json;

    constructor(string memory _path) {
        string[] memory cmds = new string[](3);
        cmds[0] = "bash";
        cmds[1] = "-c";
        cmds[2] = string.concat("cat ", _path);
        bytes memory json = vm.ffi(cmds);
        _json = string(json);
    }

    function readUint(string memory _key) external returns (uint256) {
        return stdJson.readUint(_json, _key);
    }

    function readAddress(string memory _key) external returns (address) {
        return stdJson.readAddress(_json, _key);
    }
}
