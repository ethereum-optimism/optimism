// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L2Genesis } from "scripts/L2Genesis.s.sol";
import { VmSafe } from "forge-std/Vm.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { console } from "forge-std/console.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { LibString } from "solady/utils/LibString.sol";
import { Constants } from "src/libraries/Constants.sol";

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

    function getJSONKeyCount(string memory path) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq 'keys | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    // can this become a modifier?
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

    // this is slower..
    function getBalance(string memory account) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("echo '", account, "' | jq -r '.balance'");
        return vm.parseUint(string(vm.ffi(commands)));
    }

    function getAccountCountWithNoCodeAndNoBalance(string memory path) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq 'map_values(select(.nonce == \"0x0\" and .balance == \"0x0\")) | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    // Go from keys
    function getCode(string memory account) internal returns (bytes memory) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("echo '", account, "' | jq -r '.code'");
        return bytes(vm.ffi(commands));
    }

    /// @notice Returns the number of accounts that contain particular code at a given path to a genesis file.
    function getCodeCount(string memory path, string memory name) internal returns (uint256) {
        bytes memory code = vm.getDeployedCode(name);
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq -r 'map_values(select(.code == \"", vm.toString(code), "\")) | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithStorage(string memory path, uint256 count) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq 'map_values(.storage | select(length == ", vm.toString(count), ")) | keys | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithSlotSet(string memory path, bytes32 slot) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq 'map_values(.storage | select(has(\"", vm.toString(slot), "\"))) | keys | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithSlotSetToValue(string memory path, bytes32 slot, bytes32 value) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        // jq 'map_values(.storage | select(."0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103" == "0x0000000000000000000000004200000000000000000000000000000000000018"))'
        commands[2] = string.concat("jq 'map_values(.storage | select(.\"", vm.toString(slot), "\" == \"", vm.toString(value), "\")) | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getImplementationAtAPath(string memory path, address addr) internal returns (address) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq -r '.\"", vm.toString(addr), "\".storage.\"0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc\"' < ", path);
        return address(uint160(uint256(abi.decode(vm.ffi(commands), (bytes32)))));
    }

    function testPredeployProxies() external {
        withTempDump(_testPredeployProxies);
    }

    // TODO: there are 2 addresses that dont work
    function _testPredeployProxies(string memory path) internal {
        // Set the predeploy proxies into state
        genesis.setPredeployProxies();
        genesis.writeStateDump();

        // 2 predeploys do not have proxies
        assertEq(getCodeCount(path, "Proxy.sol:Proxy"), genesis.PREDEPLOY_COUNT() - 2);

        // 17 proxies have the implementation set
        assertEq(getPredeployCountWithSlotSet(path, Constants.PROXY_IMPLEMENTATION_ADDRESS), 17);

        // All proxies except 2 have the proxy 1967 admin slot set to the proxy admin
        assertEq(getPredeployCountWithSlotSetToValue(path, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)))), genesis.PREDEPLOY_COUNT() - 2);

        // For each predeploy
        assertEq(getImplementationAtAPath(path, Predeploys.L2_TO_L1_MESSAGE_PASSER), 0xC0D3C0d3C0d3c0d3C0d3C0D3c0D3c0d3c0D30016);
        assertEq(getImplementationAtAPath(path, Predeploys.L2_CROSS_DOMAIN_MESSENGER), 0xC0d3c0d3c0D3c0D3C0d3C0D3C0D3c0d3c0d30007);
        assertEq(getImplementationAtAPath(path, Predeploys.L2_STANDARD_BRIDGE), 0xC0d3c0d3c0D3c0d3C0D3c0D3C0d3C0D3C0D30010);
        assertEq(getImplementationAtAPath(path, Predeploys.L2_ERC721_BRIDGE), 0xC0D3c0d3c0d3c0d3c0D3C0d3C0D3C0D3c0d30014);
        assertEq(getImplementationAtAPath(path, Predeploys.SEQUENCER_FEE_WALLET), 0xC0D3C0d3c0d3c0d3C0D3c0d3C0D3c0d3c0D30011);
        assertEq(getImplementationAtAPath(path, Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY), 0xc0D3c0d3C0d3c0d3c0D3c0d3c0D3c0D3c0D30012);
        assertEq(getImplementationAtAPath(path, Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY), 0xc0d3C0d3C0d3C0d3C0d3c0d3C0D3C0d3C0D30017);
        assertEq(getImplementationAtAPath(path, Predeploys.L1_BLOCK_ATTRIBUTES), 0xc0d3C0D3C0D3c0D3C0D3C0d3C0D3c0D3c0d30015);
        //assertEq(getImplementationAtAPath(path, Predeploys.GAS_PRICE_ORACLE), 0xc0d3C0d3C0d3c0D3C0D3C0d3C0d3C0D3C0D3000f);
        assertEq(getImplementationAtAPath(path, Predeploys.DEPLOYER_WHITELIST), 0xc0d3c0d3C0d3c0D3c0d3C0D3c0d3C0d3c0D30002);
        assertEq(getImplementationAtAPath(path, Predeploys.L1_BLOCK_NUMBER), 0xC0D3C0d3C0D3c0D3C0d3c0D3C0d3c0d3C0d30013);
        assertEq(getImplementationAtAPath(path, Predeploys.LEGACY_MESSAGE_PASSER), 0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000);
        assertEq(getImplementationAtAPath(path, Predeploys.PROXY_ADMIN), 0xC0d3C0D3c0d3C0d3c0d3c0D3C0D3C0d3C0D30018);
        assertEq(getImplementationAtAPath(path, Predeploys.BASE_FEE_VAULT), 0xC0d3c0D3c0d3C0D3C0D3C0d3c0D3C0D3c0d30019);
        //assertEq(getImplementationAtAPath(path, Predeploys.L1_FEE_VAULT), 0xc0D3c0D3C0d3c0d3c0d3C0d3c0d3C0d3C0D3001A);
        assertEq(getImplementationAtAPath(path, Predeploys.SCHEMA_REGISTRY), 0xc0d3c0d3c0d3C0d3c0d3C0D3C0D3c0d3C0D30020);
        assertEq(getImplementationAtAPath(path, Predeploys.EAS), 0xC0D3c0D3C0d3c0D3c0D3C0D3c0D3c0d3c0d30021);
    }
}
