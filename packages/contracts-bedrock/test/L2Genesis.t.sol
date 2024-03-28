// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L2Genesis } from "scripts/L2Genesis.s.sol";
import { VmSafe } from "forge-std/Vm.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console } from "forge-std/console.sol";
import { stdJson } from "forge-std/StdJson.sol";

contract L2GenesisTest is Test {
    L2Genesis genesis;

    function setUp() public {
        vm.setEnv("CONTRACT_ADDRESSES_PATH", string.concat(vm.projectRoot(), "/test/mocks/addresses.json"));

        genesis = new L2Genesis();
        genesis.setUp();
    }

    function tmpfile() internal returns (string memory) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = "mktemp";
        bytes memory result = vm.ffi(commands);
        return string(result);
    }

    function deleteFile(string memory path) internal {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("rm ", path);
        vm.ffi(commands);
    }

    function readJSON(string memory path) internal returns (string memory) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq < ", path);
        return string(vm.ffi(commands));
    }

    function withTempDump(function (string memory) internal f) internal {
        string memory path = tmpfile();
        vm.setEnv("STATE_DUMP_PATH", path);
        f(path);
        deleteFile(path);
    }

    function getAccount(string memory path, string memory key) internal returns (string memory) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq '.[\"", key, "\"]' < ", path);
        return string(vm.ffi(commands));
    }

    function testPredeployProxies() external {
        withTempDump(_testPredeployProxies);
    }

    function _testPredeployProxies(string memory path) internal {
        genesis.setPredeployProxies();
        genesis.writeStateDump();

        string memory dump = readJSON(path);
        string[] memory keys = vm.parseJsonKeys(dump, "");

        // 2 predeploys do not have proxies
        assertEq(keys.length, genesis.PREDEPLOY_COUNT() - 2);

        bytes memory proxyCode = vm.getDeployedCode("Proxy.sol:Proxy");
        bytes memory governanceTokenCode = vm.getDeployedCode("GovernanceToken.sol:GovernanceToken");
        bytes memory weth9Code = vm.getDeployedCode("WETH9.sol:WETH9");

        for (uint256 i; i < 60; i++) {
            string memory key = keys[i];
            string memory account = getAccount(path, key);

            // Predeploys have no balance
            uint256 balance = stdJson.readUint(account, string.concat("$.balance"));
            assertEq(balance, 0);

            // Predeploys have a nonce of 0
            uint256 nonce = stdJson.readUint(account, string.concat("$.nonce"));
            assertEq(nonce, 0);

            // Check that the bytecode + storage is correct
            address addr = vm.parseAddress(key);
            if (addr != Predeploys.GOVERNANCE_TOKEN && addr != Predeploys.WETH9) {
                // If its not an account that is explicitly not proxied, the code should be the proxy code
                assertEq(stdJson.readBytes(account, string.concat("$.code")), proxyCode);

                // All proxies has the eip 1967 admin slot set to the proxy admin
                assertEq(stdJson.readBytes32(account, string.concat("$.storage.", vm.toString(bytes32(0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103)))), bytes32(uint256(uint160(Predeploys.PROXY_ADMIN))));

                uint256 slotCount = vm.parseJsonKeys(account, string.concat("$.storage")).length;
                assertTrue(slotCount == 2 || slotCount == 1);
                if (slotCount == 2) {
                    // The other slot is the code addr
                    assertEq(stdJson.readBytes32(account, string.concat("$.storage.", vm.toString(bytes32(0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc)))), bytes32(uint256(uint160(genesis.predeployToCodeNamespace(addr)))));
                }
            } else if (addr == Predeploys.GOVERNANCE_TOKEN) {
                assertEq(stdJson.readBytes(account, string.concat("$.code")), governanceTokenCode);
            } else if (addr == Predeploys.WETH9) {
                assertEq(stdJson.readBytes(account, string.concat("$.code")), weth9Code);
            }
        }
    }
}
