// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { stdJson } from "forge-std/StdJson.sol";
import { console2 as console } from "forge-std/console2.sol";
import { VmSafe } from "forge-std/Vm.sol";

import { DeployConfig } from "scripts/DeployConfig.s.sol";
import { L2GenesisHelpers } from "scripts/libraries/L2GenesisHelpers.sol";
import { Executables } from "scripts/Executables.sol";
import { L2Genesis } from "scripts/L2Genesis.s.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { Proxy } from "src/universal/Proxy.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

/// @notice Reads a `genesis-l2.json` file, parses the `alloc`s, and runs assertions
///         against each alloc depending on whether it's a precompile, predeploy proxy,
///         or predeploy implementation.
contract L2Genesis_Test is Test, L2Genesis {
    struct StorageData {
        bytes32 key;
        bytes32 value;
    }

    struct StorageItem {
        bytes _type;
        bytes slot;
    }

    /// @custom:attribution https://github.com/Arachnid/solidity-stringutils
    struct slice {
        uint _len;
        uint _ptr;
    }

    mapping (address => mapping(bytes32 => bool)) internal expectedStorageSlots;

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

    function setUp() public override {
        L2Genesis.setUp();

        if (!vm.exists(outfilePath)) {
            run();
        }
    }

    function test_allocs() external {
        Alloc[] memory allocs = _parseAllocs(outfilePath);

        for (uint256 i; i < allocs.length; i++) {
            uint160 numericAddress = uint160(allocs[i].addr);
            if (numericAddress < L2GenesisHelpers.PRECOMPILE_COUNT) {
                _checkPrecompile(allocs[i]);
            } else if (_isProxyAddress(allocs[i].addr)) {
                // console.log(i, allocs[i].addr);
                // _checkProxy(allocs[i]);
            } else if (_isImplementationAddress(allocs[i].addr)) {
                // _checkImplementation(allocs[i]);
            } else if (_isDevAccount(allocs[i].addr)) {
                // _checkDevAccount(allocs[i]);
            } else {
                revert(string.concat("Unknown alloc: ", vm.toString(allocs[i].addr)));
            }
        }

        _checkProxy(allocs[282]);
    }

    /// @notice Runs checks against `_alloc` to determine if it's an expected precompile.
    ///         The following should hold true for every precompile:
    ///         1. The alloc should have a balance of `1`.
    ///         2. The alloc should not have `code` set.
    ///         3. The alloc should have a `nonce` of `0`.
    ///         4. The alloc should not have any storage slots set.
    function _checkPrecompile(Alloc memory _alloc) internal {
        assertEq(_alloc.balance, hex"01");
        assertEq(_alloc.code, hex"");
        assertEq(_alloc.nonce, hex"00");
        assertEq(_alloc.storageData.length, 0);
    }

    /// @notice Runs checks against `_alloc` to determine if it's an expected predeploy proxy.
    ///         The following should hold true for every predeploy proxy:
    ///         1. The alloc should have a balance of `0`.
    ///         2. The alloc should have `code` set `Proxy.sol` deployed bytecode.
    ///         3. The alloc should have a `nonce` of `0`.
    ///         4. The alloc should two storage slots set:
    ///            1. L2GenesisHelpers.PROXY_ADMIN_ADDRESS
    ///            2. L2GenesisHelpers.PROXY_IMPLEMENTATION_ADDRESS
    function _checkProxy(Alloc memory _alloc) internal {
        assertEq(_alloc.balance, hex"00");
        assertEq(_alloc.nonce, hex"00");
        _checkProxyCode(_alloc);
        _checkProxyStorage(_alloc);
    }

    // function _checkPredeployProxy(VmSafe.AccountAccess memory _access) internal {
    //     // assertEq(_access.account.balance, 0);
    //     // assertEq(vm.getNonce(_access.account), 0);

    //     // if (_isProxyAddress(_access.account)) {
    //     //     assertEq(_access.account.code, vm.getDeployedCode("Proxy.sol:Proxy"));
    //     //     assertEq(_access.storageAccesses.length, 1);
    //     //     console.logBytes32(_access.storageAccesses[0].slot);
    //     //     // assertEq(
    //     //     //     _access.storageAccesses[0].slot,
    //     //     //     EIP1967Helper.PROXY_OWNER_ADDRESS
    //     //     // );
    //     //     // assertEq(
    //     //     //     _access.storageAccesses[0].newValue,
    //     //     //     bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)))
    //     //     // );
    //     // } else if (_isImplementationAddress(_access.account)) {

    //     // } else {
    //     //     revert(string.concat("Unknown predeploy proxy: ", vm.toString(_access.account)));
    //     // }


    //     // assertEq(_access.account.code, vm.getDeployedCode("Proxy.sol:Proxy"));
    //     // // assertEq(_access.storageAccesses.length, 0);

    //     // console.log("length", _access.storageAccesses.length);
    //     // for (uint256 j; j < _access.storageAccesses.length; j++) {
    //     //     console.logBytes32(_access.storageAccesses[j].slot);
    //     //     console.logBytes32(_access.storageAccesses[j].newValue);
    //     // }
    // }

    function _addressHasPrefix(address _addr, uint160 _prefix, uint160 _mask) internal pure returns (bool) {
        uint160 numericAddress = uint160(_addr);
        return (numericAddress & _mask) == _prefix;
    }

    /// @notice Returns whether a given address has the expected predeploy proxy prefix.
    function _isProxyAddress(address _addr) internal pure returns (bool) {
        return _addressHasPrefix(
            _addr,
            uint160(0x4200000000000000000000000000000000000000),
            uint160(0xfFFFFfffFFFfFfFFffFFFfFfffFFfFfF00000000)
        );
    }

    /// @notice Returns whether a given address has the expected predeploy implementation prefix.
    function _isImplementationAddress(address _addr) internal pure returns (bool) {
        return _addressHasPrefix(
            _addr,
            uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000),
            uint160(0xfFfffFFFfffFFfFFFFffFFFFffffFfFFFFff0000)
        );
    }

    function _isDevAccount(address _addr) internal pure returns (bool) {
        address[10] memory devAccounts = abi.decode(L2GenesisHelpers.devAccountsEncoded, (address[10]));
        for (uint256 i; i < devAccounts.length; i++) {
            if (_addr == devAccounts[i]) return true;
        }
        return false;
    }

    /// @notice Parses a given `_filePath` into a `Alloc[]`.
    function _parseAllocs(string memory _filePath) internal returns (Alloc[] memory) {
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

    function _parseStorageItems(string memory _jsonString) internal returns (StorageItem[] memory) {
        string[] memory cmd = new string[](3);
        cmd[0] = "bash";
        cmd[1] = "-c";
        cmd[2] = string.concat(
            Executables.echo,
            " '",
            _jsonString,
            " ' | ",
            Executables.jq,
            " -cr '.storage | map({ slot: .slot, _type: .type })'"
        );
        bytes memory result = vm.ffi(cmd);
        bytes memory parsedJson = vm.parseJson(string(result));
        return abi.decode(parsedJson, (StorageItem[]));
    }

    /// @notice Removes the semantic versioning from a contract name. The semver will exist if the contract is compiled
    /// more than once with different versions of the compiler.
    function _stripSemver(string memory _name) internal returns (string memory) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(
            Executables.echo, " ", _name, " | ", Executables.sed, " -E 's/[.][0-9]+\\.[0-9]+\\.[0-9]+//g'"
        );
        bytes memory res = vm.ffi(cmd);
        return string(res);
    }

    function _getForgeArtifactDirectory(string memory _name) internal returns (string memory dir_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.forge, " config --json | ", Executables.jq, " -r .out");
        bytes memory res = vm.ffi(cmd);
        string memory contractName = _stripSemver(_name);
        dir_ = string.concat(vm.projectRoot(), "/", string(res), "/", contractName, ".sol");
    }

    /// @notice Returns the filesystem path to the artifact path. If the contract was compiled
    ///         with multiple solidity versions then return the first one based on the result of `ls`.
    function _getForgeArtifactPath(string memory _name) internal returns (string memory) {
        string memory directory = _getForgeArtifactDirectory(_name);
        string memory path = string.concat(directory, "/", _name, ".json");
        if (vm.exists(path)) return path;

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
        bytes memory res = vm.ffi(cmd);
        string[] memory files = stdJson.readStringArray(string(res), "");
        return string.concat(directory, "/", files[0]);
    }

    /// @notice Returns the storage layout for a deployed contract.
    function _getStorageLayout(string memory _name) internal returns (string memory layout_) {
        string[] memory cmd = new string[](3);
        cmd[0] = Executables.bash;
        cmd[1] = "-c";
        cmd[2] = string.concat(Executables.jq, " -r '.storageLayout' < ", _getForgeArtifactPath(_name));
        bytes memory res = vm.ffi(cmd);
        layout_ = string(res);
    }

    function _ignoreSlotType(string memory _type) internal pure returns (bool) {
        if (startsWith(toSlice(_type), toSlice("t_mapping"))) {
            return true;
        }

        return false;
    }

    /// @custom:attribution https://github.com/Arachnid/solidity-stringutils
    /// @dev Returns a slice containing the entire string.
    /// @param self The string to make a slice from.
    /// @return A newly allocated slice containing the entire string.
    function toSlice(string memory self) internal pure returns (slice memory) {
        uint ptr;
        assembly {
            ptr := add(self, 0x20)
        }
        return slice(bytes(self).length, ptr);
    }

    /// @custom:attribution https://github.com/Arachnid/solidity-stringutils
    /// @dev Returns true if `self` starts with `needle`.
    /// @param self The slice to operate on.
    /// @param needle The slice to search for.
    /// @return True if the slice starts with the provided text, false otherwise.
    function startsWith(slice memory self, slice memory needle) internal pure returns (bool) {
        if (self._len < needle._len) {
            return false;
        }

        if (self._ptr == needle._ptr) {
            return true;
        }

        bool equal;
        assembly {
            let length := mload(needle)
            let selfptr := mload(add(self, 0x20))
            let needleptr := mload(add(needle, 0x20))
            equal := eq(keccak256(selfptr, length), keccak256(needleptr, length))
        }
        return equal;
    }

    function _checkProxyCode(Alloc memory _alloc) internal {
        if (_alloc.addr == Predeploys.WETH9) {
            assertEq(_alloc.code, vm.getDeployedCode("WETH9.sol:WETH9"));
        } else if (_alloc.addr == Predeploys.GOVERNANCE_TOKEN) {
            // TODO Doesn't match because contract contains immutables
            // assertEq(_alloc.code, vm.getDeployedCode("GovernanceToken.sol:GovernanceToken"));
            assertNotEq(_alloc.code.length, 0);
        } else {
            assertEq(_alloc.code, vm.getDeployedCode("Proxy.sol:Proxy"));
        }
    }

    function _checkProxyStorage(Alloc memory _alloc) internal {
        string memory proxyName = getName(_alloc.addr);
        if (keccak256(abi.encodePacked(proxyName)) != keccak256("")) {
            // TODO Contract deprecated but should still check proxy admin slot still set
            if (_alloc.addr == Predeploys.L1_MESSAGE_SENDER) return;

            // TODO Are these supposed to have implementations set?
            if (_alloc.addr == Predeploys.EAS) return;
            if (_alloc.addr == Predeploys.BASE_FEE_VAULT) return;
            if (_alloc.addr == Predeploys.L1_FEE_VAULT) return;

            string memory storageLayout = _getStorageLayout(proxyName);
            StorageItem[] memory storageItems = _parseStorageItems(storageLayout);

            /// An implemeneted proxy has at least 2 storage slots set.
            /// 1. `EIP1967Helper.PROXY_OWNER_ADDRESS`
            /// 2. `EIP1967Helper.PROXY_IMPLEMENTATION_ADDRESS`
            uint256 totalNumExpectedSlots = 2;
            for (uint256 i; i < storageItems.length; i++) {
                if (_ignoreSlotType(string(storageItems[i]._type))) continue;

                totalNumExpectedSlots = ++totalNumExpectedSlots;
                expectedStorageSlots[_alloc.addr][bytes32(storageItems[i].slot)] = true;
            }

            // console.log(proxyName, _alloc.addr);

            // assertEq(
            //     totalNumExpectedSlots,
            //     _alloc.storageData.length
            // );

            for (uint256 j; j < _alloc.storageData.length; j++) {
                console.logBytes32(_alloc.storageData[j].key);
                if (_alloc.storageData[j].key == EIP1967Helper.PROXY_OWNER_ADDRESS) {
                    assertEq(
                        _alloc.storageData[j].value,
                        bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)))
                    );
                } else if (_alloc.storageData[j].key == EIP1967Helper.PROXY_IMPLEMENTATION_ADDRESS) {
                    assertEq(
                        _alloc.storageData[j].value,
                        bytes32(uint256(uint160(L2GenesisHelpers.predeployToCodeNamespace(_alloc.addr))))
                    );
                } else {
                    assertTrue(
                        expectedStorageSlots[_alloc.addr][_alloc.storageData[j].key]
                    );
                    assertEq(
                        vm.load(_alloc.addr, _alloc.storageData[j].key),
                        _alloc.storageData[j].value
                    );


                }
            }
        } else {
            /// These are unset proxies, so the slot set is the admin slot.
            assertEq(_alloc.storageData.length, 1);
            assertEq(_alloc.storageData[0].key, EIP1967Helper.PROXY_OWNER_ADDRESS);
        }
    }
}
