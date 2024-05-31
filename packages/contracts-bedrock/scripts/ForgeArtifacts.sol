// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Vm } from "forge-std/Vm.sol";
import { Executables } from "scripts/Executables.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Process } from "scripts/libraries/Process.sol";

/// @notice Contains information about a storage slot. Mirrors the layout of the storage
///         slot object in Forge artifacts so that we can deserialize JSON into this struct.
struct StorageSlot {
    uint256 astId;
    string _contract;
    string label;
    uint256 offset;
    string slot;
    string _type;
}

struct AbiEntry {
    string fnName;
    bytes4 sel;
}

struct Abi {
    string contractName;
    AbiEntry[] entries;
}

/// @title ForgeArtifacts
/// @notice Library for interacting with the forge artifacts.
library ForgeArtifacts {
    /// @notice Foundry cheatcode VM.
    Vm private constant vm = Vm(address(uint160(uint256(keccak256("hevm cheat code")))));

    /// @notice Removes the semantic versioning from a contract name. The semver will exist if the contract is compiled
    /// more than once with different versions of the compiler.
    function _stripSemver(string memory _name) internal returns (string memory out_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(
            Executables.echo, " ", _name, " | ", Executables.sed, " -E 's/[.][0-9]+\\.[0-9]+\\.[0-9]+//g'"
        );
        bytes memory res = Process.run(cmd);
        out_ = string(res);
    }

    /// @notice Builds the fully qualified name of a contract. Assumes that the
    ///         file name is the same as the contract name but strips semver for the file name.
    function _getFullyQualifiedName(string memory _name) internal returns (string memory out_) {
        string memory sanitized = _stripSemver(_name);
        out_ = string.concat(sanitized, ".sol:", _name);
    }

    /// @notice Returns the storage layout for a deployed contract.
    function getStorageLayout(string memory _name) public returns (string memory layout_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.jq, " -r '.storageLayout' < ", _getForgeArtifactPath(_name));
        bytes memory res = Process.run(cmd);
        layout_ = string(res);
    }

    /// @notice Returns the abi from a the forge artifact
    function getAbi(string memory _name) public returns (string memory abi_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.jq, " -r '.abi' < ", _getForgeArtifactPath(_name));
        bytes memory res = Process.run(cmd);
        abi_ = string(res);
    }

    /// @notice Returns the methodIdentifiers from the forge artifact
    function getMethodIdentifiers(string memory _name) internal returns (string[] memory ids_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.jq, " '.methodIdentifiers | keys' < ", _getForgeArtifactPath(_name));
        bytes memory res = Process.run(cmd);
        ids_ = stdJson.readStringArray(string(res), "");
    }

    function _getForgeArtifactDirectory(string memory _name) internal returns (string memory dir_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.forge, " config --json | ", Executables.jq, " -r .out");
        bytes memory res = Process.run(cmd);
        string memory contractName = _stripSemver(_name);
        dir_ = string.concat(vm.projectRoot(), "/", string(res), "/", contractName, ".sol");
    }

    /// @notice Returns the filesystem path to the artifact path. If the contract was compiled
    ///         with multiple solidity versions then return the first one based on the result of `ls`.
    function _getForgeArtifactPath(string memory _name) internal returns (string memory out_) {
        string memory directory = _getForgeArtifactDirectory(_name);
        string memory path = string.concat(directory, "/", _name, ".json");
        if (vm.exists(path)) {
            return path;
        }

        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(
            Executables.ls,
            " -1 --color=never ",
            directory,
            " | ",
            Executables.jq,
            " -R -s -c 'split(\"\n\") | map(select(length > 0))'"
        );
        bytes memory res = Process.run(cmd);
        string[] memory files = stdJson.readStringArray(string(res), "");
        out_ = string.concat(directory, "/", files[0]);
    }

    /// @notice Returns the forge artifact given a contract name.
    function _getForgeArtifact(string memory _name) internal returns (string memory out_) {
        string memory forgeArtifactPath = _getForgeArtifactPath(_name);
        out_ = vm.readFile(forgeArtifactPath);
    }

    /// @notice Pulls the `_initialized` storage slot information from the Forge artifacts for a given contract.
    function getInitializedSlot(string memory _contractName) internal returns (StorageSlot memory slot_) {
        string memory storageLayout = getStorageLayout(_contractName);

        string[] memory command = new string[](3);
        command[0] = Executables.bash;
        command[1] = "-c";
        command[2] = string.concat(
            Executables.echo,
            " '",
            storageLayout,
            "'",
            " | ",
            Executables.jq,
            " '.storage[] | select(.label == \"_initialized\" and .type == \"t_uint8\")'"
        );
        bytes memory rawSlot = vm.parseJson(string(Process.run(command)));
        slot_ = abi.decode(rawSlot, (StorageSlot));
    }

    /// @notice Returns whether or not a contract is initialized.
    ///         Needs the name to get the storage layout.
    function isInitialized(string memory _name, address _address) internal returns (bool initialized_) {
        StorageSlot memory slot = ForgeArtifacts.getInitializedSlot(_name);
        bytes32 slotVal = vm.load(_address, bytes32(vm.parseUint(slot.slot)));
        initialized_ = uint8((uint256(slotVal) >> (slot.offset * 8)) & 0xFF) != 0;
    }

    /// @notice Returns the function ABIs of all L1 contracts.
    function getContractFunctionAbis(
        string memory path,
        string[] memory pathExcludes
    )
        internal
        returns (Abi[] memory abis_)
    {
        string memory pathExcludesPat;
        for (uint256 i = 0; i < pathExcludes.length; i++) {
            pathExcludesPat = string.concat(pathExcludesPat, " -path ", pathExcludes[i]);
            if (i != pathExcludes.length - 1) {
                pathExcludesPat = string.concat(pathExcludesPat, " -o ");
            }
        }

        string[] memory command = new string[](3);
        command[0] = Executables.bash;
        command[1] = "-c";
        command[2] = string.concat(
            Executables.find,
            " ",
            path,
            bytes(pathExcludesPat).length > 0 ? string.concat(" ! \\( ", pathExcludesPat, " \\)") : "",
            " -type f ",
            "-exec basename {} \\;",
            " | ",
            Executables.sed,
            " 's/\\.[^.]*$//'",
            " | ",
            Executables.jq,
            " -R -s 'split(\"\n\")[:-1]'"
        );
        string[] memory contractNames = abi.decode(vm.parseJson(string(Process.run(command))), (string[]));

        abis_ = new Abi[](contractNames.length);

        for (uint256 i; i < contractNames.length; i++) {
            string memory contractName = contractNames[i];
            string[] memory methodIdentifiers = getMethodIdentifiers(contractName);
            abis_[i].contractName = contractName;
            abis_[i].entries = new AbiEntry[](methodIdentifiers.length);
            for (uint256 j; j < methodIdentifiers.length; j++) {
                string memory fnName = methodIdentifiers[j];
                bytes4 sel = bytes4(keccak256(abi.encodePacked(fnName)));
                abis_[i].entries[j] = AbiEntry({ fnName: fnName, sel: sel });
            }
        }
    }

    /// @notice Accepts a filepath and then ensures that the directory
    ///         exists for the file to live in.
    function ensurePath(string memory _path) internal {
        string[] memory outputs = vm.split(_path, "/");
        string memory path = "";
        for (uint256 i = 0; i < outputs.length - 1; i++) {
            path = string.concat(path, outputs[i], "/");
        }
        vm.createDir(path, true);
    }
}
