// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console2 as console } from "forge-std/console2.sol";
import { Strings } from "@openzeppelin/contracts/utils/Strings.sol";

import { Artifacts } from "scripts/Artifacts.s.sol";
import { Executables } from "scripts/Executables.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L2GenesisFixtures } from "test/fixtures/L2GenesisFixtures.sol";

contract L2Genesis_Test is Test, Artifacts {
    uint256 constant PROXY_COUNT = 2048;
    uint256 constant PRECOMPILE_COUNT = 256;

    string internal genesisPath;
    string[] allocAddresses;
    L2GenesisFixtures l2GenesisFixtures;

    struct StorageData {
        bytes32 key;
        bytes32 value;
    }

    /// @notice `balance` and `nonce` are being parsed as `bytes` even though their JSON representations are hex strings.
    ///         This is because Foundry has a limitation around parsing strings as numbers when using `vm.parseJson`,
    ///         and because we're using `abi.decode` to convert the JSON string, we can't use coersion (i.e. `vm.parseJsonUint`)
    ///         to tell Foundry that the strings are numbers. So instead we treat them as `byte` strings and parse as
    ///         `uint`s when needed. Additional context: https://github.com/foundry-rs/foundry/issues/3754
    struct Alloc {
        address addr;
        bytes balance;
        bytes code;
        bytes nonce;
        StorageData[] storageData;
    }

    mapping(address => StorageData[]) internal expectedAllocStorage;

    function setUp() public override {
        super.setUp();

        Artifacts.setUp();
        genesisPath = string.concat(vm.projectRoot(), "/deployments/", deploymentContext, "/genesis-l2.json");

        l2GenesisFixtures = new L2GenesisFixtures();
    }

    function test_allocs() external {
        Alloc[] memory allocs = _parseAllocs(genesisPath);

        for(uint256 i; i < allocs.length; i++) {
            uint160 numericAddress = uint160(allocs[i].addr);
            if (numericAddress < PRECOMPILE_COUNT) {
                _checkPrecompile(allocs[i]);
            } else if (_isProxyAddress(allocs[i].addr)) {
                _checkProxy(allocs[i]);
            } else if (_isImplementationAddress(allocs[i].addr)) {
                // console.log("Implementation: %s", allocs[i].addr);
            } else {
                revert(string.concat("Unknown alloc: ", Strings.toHexString(allocs[i].addr)));
            }
        }
    }

    function _checkPrecompile(Alloc memory _alloc) internal {
        assertEq(_alloc.balance, hex'01');
        assertEq(_alloc.code, hex'');
        assertEq(_alloc.nonce, hex'00');
        assertEq(_alloc.storageData.length, 0);
    }

    function _checkProxy(Alloc memory _alloc) internal {
        assertEq(_alloc.balance, hex'00');
        assertEq(_alloc.nonce, hex'00');

        if (!_notProxied(_alloc.addr)) {
            assertEq(_alloc.code, vm.getDeployedCode("Proxy.sol:Proxy"));
        }

        /// By asserting the length of the `_alloc.storageData` and `expectedStorageData` are equal, we know
        /// that we at least have the expected amount of storage slots and their corresponding values (i.e. we don't have missing or extra storage slots).
        /// Next we assert that the `value` of each `_alloc.storageData`'s `key` matches the expected value for that slot.
        /// After these assertions, we now know that for this `_alloc`, we have the expected number of storage slot keys, and the correct corrsponding values.
        /// However, this misses the edgecase where, for example, `_alloc.storageData.length` equals 2 and `expectedStorageData.length` equals 2, but the keys
        /// set in ``_alloc.storageData` are both 0x000...000 and their values are 0x000.000, so when we look up the corresponding values using
        /// `l2GenesisFixtures.getStorageValueBySlot`, if the expected values for those keys are their default mapping values, the test assertions will pass, even if
        /// `expectedStorageData` is expected different keys. To solve this we add an assertion for `l2GenesisFixtures.slotIsInitialized`, to verify the slot is supposed
        /// to be set even if the value is 0x000...000.
        L2GenesisFixtures.StorageData[] memory expectedStorageData = l2GenesisFixtures.getStorageData(_alloc.addr);
        assertEq(_alloc.storageData.length, expectedStorageData.length);
        for (uint256 i; i < expectedStorageData.length; i++) {
            assertEq(true, l2GenesisFixtures.slotIsInitialized(_alloc.addr, _alloc.storageData[i].key));
            assertEq(
                _alloc.storageData[i].value,
                l2GenesisFixtures.getStorageValueBySlot(_alloc.addr, _alloc.storageData[i].key)
            );
        }
    }

    function _parseAllocs(string memory _filePath) internal returns(Alloc[] memory) {
        string[] memory cmd = new string[](3);
        cmd[0] = "bash";
        cmd[1] = "-c";
        cmd[2] = string.concat(
            Executables.jq,
            " -cr 'to_entries | map({addr: .key, balance: .value.balance, code: .value.code, nonce: .value.nonce, storageData: (.value.storage | to_entries | map({key: .key, value: .value}))})' ",
            _filePath
        );
        bytes memory result = vm.ffi(cmd);
        bytes memory parsedJson = vm.parseJson(string(result));
        return abi.decode(parsedJson, (Alloc[]));
    }

    function _addressHasPrefix(address _addr, uint160 _prefix, uint160 _mask) internal pure returns(bool) {
        uint160 numericAddress = uint160(_addr);
        return (numericAddress & _mask) == _prefix;
    }

    function _isProxyAddress(address _addr) internal pure returns(bool) {
        return _addressHasPrefix(
            _addr,
            uint160(0x4200000000000000000000000000000000000000),
            uint160(0xfFFFFfffFFFfFfFFffFFFfFfffFFfFfF00000000)
        );
    }

    function _isImplementationAddress(address _addr) internal pure returns(bool) {
        return _addressHasPrefix(
            _addr,
            uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000),
            uint160(0xfFfffFFFfffFFfFFFFffFFFFffffFfFFFFff0000)
        );
    }

    function _notProxied(address _addr) internal pure returns(bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }

    function _setExpectedAllocStorage() internal {

    }
}
