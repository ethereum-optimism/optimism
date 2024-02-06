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

    struct StorageData {
        bytes32 key;
        bytes32 value;
    }

    struct Alloc {
        address _address;
        bytes balance;
        bytes code;
        bytes nonce;
        StorageData[] storageData;
    }

    function setUp() public override {
        super.setUp();

        Artifacts.setUp();
        genesisPath = string.concat(vm.projectRoot(), "/deployments/", deploymentContext, "/genesis-l2.json");
    }

    function test_allocs() external {
        Alloc[] memory allocs = _parseAllocs(genesisPath);

        for(uint256 i; i < allocs.length; i++) {
            uint160 numericAddress = uint160(allocs[i]._address);
            if (numericAddress < PRECOMPILE_COUNT) {
                _checkPrecompile(allocs[i]);
            }
        }
    }

    function _checkPrecompile(Alloc memory alloc) internal {
        assertEq(alloc.balance, hex'01');
        assertEq(alloc.code, hex'');
        assertEq(alloc.nonce, hex'00');
        assertEq(alloc.storageData.length, 0);
    }

    function _parseAllocs(string memory filePath) internal returns(Alloc[] memory) {
        string memory jqCommand = string.concat(
            "jq -cr 'to_entries | map({address: .key, balance: .value.balance, code: .value.code, nonce: .value.nonce, storageData: (.value.storage | to_entries | map({key: .key, value: .value}))})' ",
            filePath
        );

        string[] memory cmd = new string[](3);
        cmd[0] = "bash";
        cmd[1] = "-c";
        cmd[2] = jqCommand;
        bytes memory result = vm.ffi(cmd);
        bytes memory parsedJson = vm.parseJson(string(result));
        return abi.decode(parsedJson, (Alloc[]));

        // console.log(_allocs[0]._address);
        // console.logBytes(_allocs[0].balance);
        // console.logBytes(_allocs[0].code);
        // console.logBytes(_allocs[0].nonce);
        // console.logBytes32(_allocs[0].storageData[0].key);
        // console.logBytes32(_allocs[0].storageData[0].value);
        // console.logBytes32(_allocs[0].storageData[1].key);
        // console.logBytes32(_allocs[0].storageData[1].value);
    }
}
