// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { L2Genesis, OutputMode, L1Dependencies } from "scripts/L2Genesis.s.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";

/// @title L2GenesisTest
/// @notice Test suite for L2Genesis script.
contract L2GenesisTest is Test {
    L2Genesis genesis;

    function setUp() public {
        genesis = new L2Genesis();
        // Note: to customize L1 addresses,
        // simply pass in the L1 addresses argument for Genesis setup functions that depend on it.
        // L1 addresses, or L1 artifacts, are not stored globally.
        genesis.setUp();
    }

    /// @notice Creates a temp file and returns the path to it.
    function tmpfile() internal returns (string memory) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = "mktemp";
        bytes memory result = vm.ffi(commands);
        return string(result);
    }

    /// @notice Deletes a file at a given filesystem path. Does not force delete
    ///         and does not recursively delete.
    function deleteFile(string memory path) internal {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("rm ", path);
        vm.ffi(commands);
    }

    /// @notice Returns the number of top level keys in a JSON object at a given
    ///         file path.
    function getJSONKeyCount(string memory path) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] = string.concat("jq 'keys | length' < ", path, " | xargs cast abi-encode 'f(uint256)'");
        return abi.decode(vm.ffi(commands), (uint256));
    }

    /// @notice Helper function to run a function with a temporary dump file.
    function withTempDump(function (string memory) internal f) internal {
        string memory path = tmpfile();
        f(path);
        deleteFile(path);
    }

    /// @notice Helper function for reading the number of storage keys for a given account.
    function getStorageKeysCount(string memory _path, address _addr) internal returns (uint256) {
        string[] memory commands = new string[](3);
        commands[0] = "bash";
        commands[1] = "-c";
        commands[2] =
            string.concat("jq -r '.[\"", vm.toLowercase(vm.toString(_addr)), "\"].storage | length' < ", _path);
        return vm.parseUint(string(vm.ffi(commands)));
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

    /// @notice Returns the number of accounts that have a particular slot set.
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

    /// @notice Returns the number of accounts that have a particular slot set to a particular value.
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

    /// @notice Tests the genesis predeploys setup using a temp file.
    function test_genesis_predeploys() external {
        withTempDump(_test_genesis_predeploys);
    }

    /// @notice Tests the genesis predeploys setup.
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

    /// @notice Tests the number of accounts in the genesis setup
    function test_allocs_size() external {
        withTempDump(_test_allocs_size);
    }

    /// @notice Creates mock L1Dependencies for testing purposes.
    function _dummyL1Deps() internal pure returns (L1Dependencies memory _deps) {
        return L1Dependencies({
            l1CrossDomainMessengerProxy: payable(address(0x100000)),
            l1StandardBridgeProxy: payable(address(0x100001)),
            l1ERC721BridgeProxy: payable(address(0x100002))
        });
    }

    /// @notice Tests the number of accounts in the genesis setup
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
