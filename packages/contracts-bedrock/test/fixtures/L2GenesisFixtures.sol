// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";
import { console2 as console } from "forge-std/console2.sol";

import { Predeploys } from "src/libraries/Predeploys.sol";

contract L2GenesisFixtures {
    uint256 constant PROXY_COUNT = 2048;
    /// @notice The storage slot that holds the address of the owner.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)`
    bytes32 internal constant PROXY_ADMIN_ADDRESS = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;
    /// @notice The storage slot that holds the address of a proxy implementation.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)`
    bytes32 internal constant PROXY_IMPLEMENTATION_ADDRESS =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    mapping(address => uint256) numExpectedSlotKeys;
    mapping(address => mapping(bytes32 => bool)) expectedSlotKeys;
    mapping(address => mapping(bytes32 => bytes32)) slotValueByKey;

    function setUp() public virtual {
        _setProxyStorageData();
    }

    function getNumExpectedSlotKeys(address _addr) public view returns(uint256) {
        return numExpectedSlotKeys[_addr];
    }

    function isExpectedSlotKey(address _addr, bytes32 _slot) public view returns(bool) {
        return expectedSlotKeys[_addr][_slot];
    }

    function getSlotValueByKey(address _addr, bytes32 _slot) public view returns(bytes32) {
        return slotValueByKey[_addr][_slot];
    }

    function _setProxyStorageData() internal {
        uint160 prefix = uint160(0x420) << 148;

        for (uint256 i; i < PROXY_COUNT; i++) {
            address addr = address(prefix | uint160(i));

            if (_notProxied(addr)) {
                continue;
            }

            numExpectedSlotKeys[addr] = ++numExpectedSlotKeys[addr];
            expectedSlotKeys[addr][PROXY_ADMIN_ADDRESS] = true;
            slotValueByKey[addr][PROXY_ADMIN_ADDRESS] = bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)));

            if (_hasImplementation(addr)) {
                address implementation = _predeployToCodeNamespace(addr);
                numExpectedSlotKeys[addr] = ++numExpectedSlotKeys[addr];
                expectedSlotKeys[addr][PROXY_IMPLEMENTATION_ADDRESS] = true;
                slotValueByKey[addr][PROXY_IMPLEMENTATION_ADDRESS] = bytes32(uint256(uint160(implementation)));
            }

        }

        _setWETH9StorageData();
        _setL2CrossDomainMessengerStorageData();
        _setL2StandardBridgeStorageData();
        _setOptimismMintableERC20FactoryStorageData();
        _setGovernanceTokenStorageData();
    }

    function _setWETH9StorageData() internal {
        bytes32[3] memory expectedStorageKeys = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000000),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000001),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000002)
        ];
        bytes32[3] memory expectedStorageValues = [
            bytes32(0x577261707065642045746865720000000000000000000000000000000000001a),
            bytes32(0x5745544800000000000000000000000000000000000000000000000000000008),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000012)
        ];

        _setFixtureData(Predeploys.WETH9, expectedStorageKeys, expectedStorageValues);
    }

    function _setL2CrossDomainMessengerStorageData() internal {
        bytes32[3] memory expectedStorageKeys = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000000),
            bytes32(0x00000000000000000000000000000000000000000000000000000000000000cc),
            bytes32(0x00000000000000000000000000000000000000000000000000000000000000cf)
        ];
        bytes32[3] memory expectedStorageValues = [
            bytes32(0x0000000000000000000000010000000000000000000000000000000000000000),
            bytes32(0x000000000000000000000000000000000000000000000000000000000000dead),
            bytes32(0x00000000000000000000000020a42a5a785622c6ba2576b2d6e924aa82bfa11d)
        ];

        _setFixtureData(Predeploys.L2_CROSS_DOMAIN_MESSENGER, expectedStorageKeys, expectedStorageValues);
    }

    function _setL2StandardBridgeStorageData() internal {
        bytes32[3] memory expectedStorageKeys = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000000),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000003),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000004)
        ];
        bytes32[3] memory expectedStorageValues = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000001),
            bytes32(0x0000000000000000000000004200000000000000000000000000000000000007),
            bytes32(0x0000000000000000000000000c8b5822b6e02cda722174f19a1439a7495a3fa6)
        ];

        _setFixtureData(Predeploys.L2_STANDARD_BRIDGE, expectedStorageKeys, expectedStorageValues);
    }

    function _setOptimismMintableERC20FactoryStorageData() internal {
        bytes32[2] memory expectedStorageKeys = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000000),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000001)
        ];
        bytes32[2] memory expectedStorageValues = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000001),
            bytes32(0x0000000000000000000000004200000000000000000000000000000000000010)
        ];

        _setFixtureData(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY, expectedStorageKeys, expectedStorageValues);
    }

    function _setGovernanceTokenStorageData() internal {
        bytes32[3] memory expectedStorageKeys = [
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000003),
            bytes32(0x0000000000000000000000000000000000000000000000000000000000000004),
            bytes32(0x000000000000000000000000000000000000000000000000000000000000000a)
        ];
        bytes32[3] memory expectedStorageValues = [
            bytes32(0x4f7074696d69736d000000000000000000000000000000000000000000000010),
            bytes32(0x4f50000000000000000000000000000000000000000000000000000000000004),
            bytes32(0x000000000000000000000000a0ee7a142d267c1f36714e4a8f75612f20a79720)
        ];

        _setFixtureData(Predeploys.GOVERNANCE_TOKEN, expectedStorageKeys, expectedStorageValues);
    }

    //////////////////////////////////////////////////////
    /// Helper Functions
    //////////////////////////////////////////////////////
    function _notProxied(address _addr) internal pure returns(bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }

    function _hasImplementation(address _addr) internal pure returns(bool) {
        return _addr == Predeploys.LEGACY_MESSAGE_PASSER ||
            _addr == Predeploys.DEPLOYER_WHITELIST ||
            _addr == Predeploys.L2_CROSS_DOMAIN_MESSENGER ||
            _addr == Predeploys.GAS_PRICE_ORACLE ||
            _addr == Predeploys.L2_STANDARD_BRIDGE ||
            _addr == Predeploys.SEQUENCER_FEE_WALLET ||
            _addr == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY ||
            _addr == Predeploys.L1_BLOCK_NUMBER ||
            _addr == Predeploys.L2_ERC721_BRIDGE ||
            _addr == Predeploys.L1_BLOCK_ATTRIBUTES ||
            _addr == Predeploys.L2_TO_L1_MESSAGE_PASSER ||
            _addr == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY ||
            _addr == Predeploys.PROXY_ADMIN ||
            _addr == Predeploys.BASE_FEE_VAULT ||
            _addr == Predeploys.L1_FEE_VAULT ||
            _addr == Predeploys.SCHEMA_REGISTRY ||
            _addr == Predeploys.EAS;
    }

    function _predeployToCodeNamespace(address _addr) internal pure returns (address) {
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
    }

    function _setFixtureData(address _addr, bytes32[2] memory _expectedStorageKeys, bytes32[2] memory _expectedStorageValues) internal {
        for(uint256 i; i < _expectedStorageKeys.length; i++) {
            numExpectedSlotKeys[_addr] = ++numExpectedSlotKeys[_addr];
            expectedSlotKeys[_addr][_expectedStorageKeys[i]] = true;
            slotValueByKey[_addr][_expectedStorageKeys[i]] = _expectedStorageValues[i];
        }
    }

    function _setFixtureData(address _addr, bytes32[3] memory _expectedStorageKeys, bytes32[3] memory _expectedStorageValues) internal {
        for(uint256 i; i < _expectedStorageKeys.length; i++) {
            numExpectedSlotKeys[_addr] = ++numExpectedSlotKeys[_addr];
            expectedSlotKeys[_addr][_expectedStorageKeys[i]] = true;
            slotValueByKey[_addr][_expectedStorageKeys[i]] = _expectedStorageValues[i];
        }
    }
}
