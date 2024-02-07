// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Script } from "forge-std/Script.sol";

import { Predeploys } from "src/libraries/Predeploys.sol";

contract L2GenesisFixtures is Script {
    uint256 constant PROXY_COUNT = 2048;
    /// @notice The storage slot that holds the address of the owner.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.admin')) - 1)`
    bytes32 internal constant PROXY_ADMIN_ADDRESS = 0xb53127684a568b3173ae13b9f8a6016e243e63b6e8ee1178d6a717850b5d6103;
    /// @notice The storage slot that holds the address of a proxy implementation.
    /// @dev `bytes32(uint256(keccak256('eip1967.proxy.implementation')) - 1)`
    bytes32 internal constant PROXY_IMPLEMENTATION_ADDRESS =
        0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc;

    struct StorageData {
        bytes32 key;
        bytes32 value;
    }

    mapping(address => StorageData[]) public storageDatas;
    mapping(address => mapping(bytes32 => bytes32)) public storageSlotValues;

    function setUp() public {
        _setProxyStorageData();
    }

    function getStorageData(address _addr) public view returns(StorageData[] memory) {
        return storageDatas[_addr];
    }

    function getStorageValueBySlot(address _addr, bytes32 _slot) public view returns(bytes32) {
        return storageSlotValues[_addr][_slot];
    }

    function _setProxyStorageData() internal {
        uint160 prefix = uint160(0x420) << 148;

        for (uint256 i; i < PROXY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            if (!_notProxied(addr)) {
                storageDatas[addr].push(
                    StorageData({
                        key: PROXY_ADMIN_ADDRESS,
                        value: bytes32(uint256(uint160(Predeploys.PROXY_ADMIN)))
                    })
                );


                /// L1 Message Sender has been deprecated and doesn't have the
                // PROXY_IMPLEMENTATION_ADDRESS slot set.
                if (addr != Predeploys.L1_MESSAGE_SENDER) continue;

                address implementation = _predeployToCodeNamespace(addr);
                storageDatas[addr].push(
                    StorageData({
                        key: PROXY_IMPLEMENTATION_ADDRESS,
                        value: bytes32(uint256(uint160(implementation)))
                    })
                );
            }
        }

        _setWETH9StorageData();
        _setL2CrossDomainMessengerStorageData();
        _setL2StandardBridgeStorageData();
        _setOptimismMintableERC20FactoryStorageData();
        _setGovernanceTokenStorageData();
    }

    function _notProxied(address _addr) internal pure returns(bool) {
        return _addr == Predeploys.GOVERNANCE_TOKEN || _addr == Predeploys.WETH9;
    }

    function _predeployToCodeNamespace(address _addr) internal pure returns (address) {
        return address(
            uint160(uint256(uint160(_addr)) & 0xffff | uint256(uint160(0xc0D3C0d3C0d3C0D3c0d3C0d3c0D3C0d3c0d30000)))
        );
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

        for(uint256 i; i < expectedStorageKeys.length; i++) {
            storageDatas[Predeploys.WETH9].push(
                StorageData({
                    key: expectedStorageKeys[i],
                    value: expectedStorageValues[i]
                })
            );

            storageSlotValues[Predeploys.WETH9][expectedStorageKeys[i]] = expectedStorageValues[i];
        }
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

        for(uint256 i; i < expectedStorageKeys.length; i++) {
            storageDatas[Predeploys.L2_CROSS_DOMAIN_MESSENGER].push(
                StorageData({
                    key: expectedStorageKeys[i],
                    value: expectedStorageValues[i]
                })
            );

            storageSlotValues[Predeploys.L2_CROSS_DOMAIN_MESSENGER][expectedStorageKeys[i]] = expectedStorageValues[i];
        }
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

        for(uint256 i; i < expectedStorageKeys.length; i++) {
            storageDatas[Predeploys.L2_STANDARD_BRIDGE].push(
                StorageData({
                    key: expectedStorageKeys[i],
                    value: expectedStorageValues[i]
                })
            );

            storageSlotValues[Predeploys.L2_STANDARD_BRIDGE][expectedStorageKeys[i]] = expectedStorageValues[i];
        }
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

        for(uint256 i; i < expectedStorageKeys.length; i++) {
            storageDatas[Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY].push(
                StorageData({
                    key: expectedStorageKeys[i],
                    value: expectedStorageValues[i]
                })
            );

            storageSlotValues[Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY][expectedStorageKeys[i]] = expectedStorageValues[i];
        }
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

        for(uint256 i; i < expectedStorageKeys.length; i++) {
            storageDatas[Predeploys.GOVERNANCE_TOKEN].push(
                StorageData({
                    key: expectedStorageKeys[i],
                    value: expectedStorageValues[i]
                })
            );

            storageSlotValues[Predeploys.GOVERNANCE_TOKEN][expectedStorageKeys[i]] = expectedStorageValues[i];
        }
    }
}
