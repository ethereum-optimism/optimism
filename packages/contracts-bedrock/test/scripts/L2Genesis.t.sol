// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console2 as console } from "forge-std/console2.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

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
        string memory jqCommand = string.concat(
            "jq -cr 'to_entries | map({address: .key, balance: .value.balance, code: .value.code, nonce: .value.nonce, storageKeys: (.value.storage | keys), storageValues: (.value.storage | [.[]])})' ",
            filePath
        );

        string[] memory cmd = new string[](3);
        cmd[0] = "bash";
        cmd[1] = "-c";
        cmd[2] = jqCommand;
        bytes memory result = vm.ffi(cmd);

        string memory jsonResult = string(result);
        // uint allocCount = stdJson.parseUint(stdJson.count(jsonResult, "$"));
        for (uint i = 0; i < 2313; i++) {
            string memory basePath = string.concat("$[", Strings.toString(i), "]");
            Alloc memory alloc;
            alloc.balance = stdJson.readString(jsonResult, string.concat(basePath, ".balance"));
            alloc.code = stdJson.readString(jsonResult, string.concat(basePath, ".code"));
            alloc.nonce = stdJson.readString(jsonResult, string.concat(basePath, ".nonce"));
            alloc.storageKeys = stdJson.readStringArray(jsonResult, string.concat(basePath, ".storageKeys"));
            alloc.storageValues = stdJson.readStringArray(jsonResult, string.concat(basePath, ".storageValues"));
            string memory addressStr = stdJson.readString(jsonResult, string.concat(basePath, ".address"));
            address allocAddress = vm.parseAddress(addressStr);

            allocs[allocAddress] = alloc;
        }
    }
}
