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

    // mapping(address => Alloc) allocs;

    struct Alloc {
        string balance;
        string code;
        // string nonce;
        // string[] storageKeys;
        // string[] storageValues;
    }

    Alloc[] allocs;

    function setUp() public override {
        super.setUp();

        Artifacts.setUp();
        genesisPath = string.concat(vm.projectRoot(), "/deployments/", deploymentContext, "/genesis-l2.json");
        _parseAllocs(genesisPath);
    }

    function test_allocs() external {
        // _checkPrecompiles();
    }

    function _checkPrecompiles() internal {
        // for (uint256 i; i < 1; i++) {
        //     address expectedAddress = address(uint160(i));
        //     console.log("Checking precompile: %s", expectedAddress);

        //     Alloc storage alloc = allocs[expectedAddress];
        //     assertEq(alloc.balance, "0x1");
        // }
    }

    function _parseAllocs(string memory filePath) internal {
        string memory jqCommand = string.concat(
            // "jq -cr 'to_entries | map({address: .key, balance: .value.balance, code: .value.code, nonce: .value.nonce, storageKeys: (.value.storage | keys), storageValues: (.value.storage | [.[]])})' ",
            "jq -cr 'to_entries | map({address: .key, balance: .value.balance, code: .value.code})' ",
            filePath
        );

        string[] memory cmd = new string[](3);
        cmd[0] = "bash";
        cmd[1] = "-c";
        cmd[2] = jqCommand;
        bytes memory result = vm.ffi(cmd);
        bytes memory parsedJson = vm.parseJson(string(result));
        Alloc[] memory _allocs = abi.decode(parsedJson, (Alloc[]));

        console.log(_allocs[0].balance);
        console.log(_allocs[0].code);
    }
}
