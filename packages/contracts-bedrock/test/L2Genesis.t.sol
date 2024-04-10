// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L2Genesis, OutputMode, L1Dependencies } from "scripts/L2Genesis.s.sol";
import { console2 as console } from "forge-std/console2.sol";
import { VmSafe } from "forge-std/Vm.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { Constants } from "src/libraries/Constants.sol";

contract L2GenesisTest is Test {
    L2Genesis genesis;

    function setUp() public {
        genesis = new L2Genesis();
        // Note: to customize L1 addresses,
        // simply pass in the L1 addresses argument for Genesis setup functions that depend on it.
        // L1 addresses, or L1 artifacts, are not stored globally.
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

    function withTempDump(function (string memory) internal f) internal {
        string memory path = tmpfile();
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

    function getStorageKeysCount(string memory _path, address _addr) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] =
            string.concat("jq -r '.[\"", vm.toLowercase(vm.toString(_addr)), "\"].storage | length' < ", _path);
        return vm.parseUint(string(vm.ffi(commands)));
    }

    function getAccountCountWithNoCodeAndNoBalance(string memory path) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat(
            "jq 'map_values(select(.nonce == \"0x0\" and .balance == \"0x0\")) | length' < ",
            path,
            " | xargs cast abi-encode 'f(uint256)'"
        );
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
        commands[2] = string.concat(
            "jq -r 'map_values(select(.code == \"",
            vm.toString(code),
            "\")) | length' < ",
            path,
            " | xargs cast abi-encode 'f(uint256)'"
        );
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithStorage(string memory path, uint256 count) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat(
            "jq 'map_values(.storage | select(length == ",
            vm.toString(count),
            ")) | keys | length' < ",
            path,
            " | xargs cast abi-encode 'f(uint256)'"
        );
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithSlotSet(string memory path, bytes32 slot) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat(
            "jq 'map_values(.storage | select(has(\"",
            vm.toString(slot),
            "\"))) | keys | length' < ",
            path,
            " | xargs cast abi-encode 'f(uint256)'"
        );
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getPredeployCountWithSlotSetToValue(
        string memory path,
        bytes32 slot,
        bytes32 value
    )
        internal
        returns (uint256)
    {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        // jq 'map_values(.storage | select(."0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103" ==
        // "0x0000000000000000000000004200000000000000000000000000000000000018"))'
        commands[2] = string.concat(
            "jq 'map_values(.storage | select(.\"",
            vm.toString(slot),
            "\" == \"",
            vm.toString(value),
            "\")) | length' < ",
            path,
            " | xargs cast abi-encode 'f(uint256)'"
        );
        return abi.decode(vm.ffi(commands), (uint256));
    }

    function getImplementationAtAPath(string memory path, address addr) internal returns (address) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        // Forge state dumps use lower-case addresses as keys in the allocs dictionary.
        commands[2] = string.concat(
            "jq -r '.\"",
            vm.toLowercase(vm.toString(addr)),
            "\".storage.\"0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc\"' < ",
            path
        );
        return address(uint160(uint256(abi.decode(vm.ffi(commands), (bytes32)))));
    }

    function test_genesis_predeploys() external {
        withTempDump(_test_genesis_predeploys);
    }

    function _test_genesis_predeploys(string memory _path) internal {
        // Set the predeploy proxies into state
        genesis.setPredeployProxies();
        genesis.writeGenesisAllocs(_path);

        // 2 predeploys do not have proxies
        assertEq(getCodeCount(_path, "Proxy.sol:Proxy"), Predeploys.PREDEPLOY_COUNT - 2);

        // 17 proxies have the implementation set
        assertEq(getPredeployCountWithSlotSet(_path, Constants.PROXY_IMPLEMENTATION_ADDRESS), 17);

        // All proxies except 2 have the proxy 1967 admin slot set to the proxy admin
        assertEq(
            getPredeployCountWithSlotSetToValue(
                _path, Constants.PROXY_OWNER_ADDRESS, bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)))
            ),
            Predeploys.PREDEPLOY_COUNT - 2
        );

        // Also see Predeploys.t.test_predeploysSet_succeeds which uses L1Genesis for the CommonTest prestate.
    }

    function test_allocs_size() external {
        withTempDump(_test_allocs_size);
    }

    function _dummyL1Deps() internal pure returns (L1Dependencies memory _deps) {
        return L1Dependencies({
            l1CrossDomainMessengerProxy: payable(address(0x100000)),
            l1StandardBridgeProxy: payable(address(0x100001)),
            l1ERC721BridgeProxy: payable(address(0x100002))
        });
    }

    function _test_allocs_size(string memory _path) internal {
        genesis.runWithOptions(OutputMode.LOCAL_LATEST, _dummyL1Deps());
        genesis.writeGenesisAllocs(_path);

        uint256 expected = 0;
        expected += 2048 - 2; // predeploy proxies
        expected += 19; // predeploy implementations (excl. legacy erc20-style eth and legacy message sender)
        expected += 256; // precompiles
        expected += 12; // preinstalls
        expected += 1; // 4788 deployer account
        // 16 prefunded dev accounts are excluded
        assertEq(expected, getJSONKeyCount(_path), "key count check");

        // 3 slots: implementation, owner, admin
        assertEq(3, getStorageKeysCount(_path, Predeploys.PROXY_ADMIN), "proxy admin storage check");
    }
}
