// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console2 as console } from "forge-std/console2.sol";

import { Artifacts } from "scripts/Artifacts.s.sol";
import { Executables } from "scripts/Executables.sol";

contract L2Genesis_Test is Test, Artifacts {
    uint256 constant PROXY_COUNT = 2048;
    uint256 constant PRECOMPILE_COUNT = 256;

    string internal genesisPath;
    string[] allocAddresses;

    mapping(address => Alloc) allocs;

    struct Alloc {
        string balance;
        string code;
        string nonce;
        string[] storageKeys;
        string[] storageValues;
    }

    function setUp() public override {
        super.setUp();

        Artifacts.setUp();
        genesisPath = string.concat(vm.projectRoot(), "/deployments/", deploymentContext, "/genesis-l2.json");
        _parseAllocs(genesisPath);
    }

    function test_allocs() external {
        _checkPrecompiles();
    }

    function _checkPrecompiles() internal {
        for (uint256 i; i < PRECOMPILE_COUNT; i++) {
            address expectedAddress = address(uint160(i));
            console.log("Checking precompile: %s", expectedAddress);

            Alloc storage alloc = allocs[expectedAddress];
            assertEq(alloc.balance, "0x1");
        }
    }

    function _parseAllocs(string memory filePath) internal {
        console.log("Parsing allocs");
        string[] memory getAllocAddressesCmd = new string[](3);
        getAllocAddressesCmd[0] = "bash";
        getAllocAddressesCmd[1] = "-c";
        getAllocAddressesCmd[2] = string.concat("jq 'keys' ", filePath);
        bytes memory rawAllocAddresses = vm.ffi(getAllocAddressesCmd);
        allocAddresses = stdJson.readStringArray(string(rawAllocAddresses), "");

        for (uint256 i; i < allocAddresses.length; i++) {
            Alloc memory alloc;

            bytes memory rawAllocProperties = _getAllocProperties(filePath, allocAddresses[i]);
            alloc.balance = stdJson.readString(string(rawAllocProperties), "$.balance");
            alloc.code = stdJson.readString(string(rawAllocProperties), "$.code");
            alloc.nonce = stdJson.readString(string(rawAllocProperties), "$.nonce");

            alloc.storageKeys = _getAllocStorageKeys(filePath, allocAddresses[i]);
            alloc.storageValues = _getAllocStorageValues(filePath, allocAddresses[i]);

            allocs[vm.parseAddress(allocAddresses[i])] = alloc;
        }
    }

    function _getAllocProperties(string memory filePath, string memory allocAddress) internal returns(bytes memory) {
        string[] memory getAllocPropertiesCmd = new string[](3);
        getAllocPropertiesCmd[0] = "bash";
        getAllocPropertiesCmd[1] = "-c";
        getAllocPropertiesCmd[2] = string.concat(
            Executables.jq,
            " -cr '.[\"",
            allocAddress,
            "\"]' ",
            filePath
        );
        return vm.ffi(getAllocPropertiesCmd);
    }

    function _getAllocStorageKeys(string memory filePath, string memory allocAddress) internal returns(string[] memory) {
        string[] memory getAllocStorageKeysCmd = new string[](3);
        getAllocStorageKeysCmd[0] = "bash";
        getAllocStorageKeysCmd[1] = "-c";
        getAllocStorageKeysCmd[2] = string.concat(
            Executables.jq,
            " -cr '",
            "[.\"", allocAddress, "\".storage | to_entries[] | .key]' ",
            filePath
        );
        bytes memory rawAllocStorageKeys = vm.ffi(getAllocStorageKeysCmd);
        return stdJson.readStringArray(string(rawAllocStorageKeys), "");
    }

    function _getAllocStorageValues(string memory filePath, string memory allocAddress) internal returns(string[] memory) {
        string[] memory getAllocStorageValuesCmd = new string[](3);
        getAllocStorageValuesCmd[0] = "bash";
        getAllocStorageValuesCmd[1] = "-c";
        getAllocStorageValuesCmd[2] = string.concat(
            Executables.jq,
            " -cr '",
            "[.\"", allocAddress, "\".storage | to_entries[] | .value]' ",
            filePath
        );
        bytes memory rawAllocStorageValues = vm.ffi(getAllocStorageValuesCmd);
        return stdJson.readStringArray(string(rawAllocStorageValues), "");
    }
}
