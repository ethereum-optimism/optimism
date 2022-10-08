// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";
import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L1StandardBridge } from "../L1/L1StandardBridge.sol";
import { L1ERC721Bridge } from "../L1/L1ERC721Bridge.sol";
import { OptimismMintableERC20Factory } from "../universal/OptimismMintableERC20Factory.sol";

/**
 * @title FreshSystemDictator
 * @notice The FreshSystemDictator is responsible for coordinating initialization of a fresh
 *         deployment of the Optimism system. We expect that all proxies and implementations
 *         already be deployed before this contract is used.
 */
contract FreshSystemDictator is Ownable {
    struct GlobalConfig {
        ProxyAdmin proxyAdmin;
        address controller;
        address finalOwner;
    }

    struct ProxyAddressConfig {
        address l2OutputOracleProxy;
        address optimismPortalProxy;
        address l1CrossDomainMessengerProxy;
        address l1StandardBridgeProxy;
        address optimismMintableERC20FactoryProxy;
        address l1ERC721BridgeProxy;
    }

    struct ImplementationAddressConfig {
        L2OutputOracle l2OutputOracleImpl;
        OptimismPortal optimismPortalImpl;
        L1CrossDomainMessenger l1CrossDomainMessengerImpl;
        L1StandardBridge l1StandardBridgeImpl;
        OptimismMintableERC20Factory optimismMintableERC20FactoryImpl;
        L1ERC721Bridge l1ERC721BridgeImpl;
    }

    struct L2OutputOracleConfig {
        bytes32 l2OutputOracleGenesisL2Output;
        address l2OutputOracleProposer;
        address l2OutputOracleOwner;
    }

    struct Config {
        GlobalConfig globalConfig;
        ProxyAddressConfig proxyAddressConfig;
        ImplementationAddressConfig implementationAddressConfig;
        L2OutputOracleConfig l2OutputOracleConfig;
    }

    Config public config;

    constructor(Config memory _config) Ownable() {
        config = _config;
        _transferOwnership(config.globalConfig.controller);
    }

    function step1() external onlyOwner {
        // Upgrade and initialize the L2OutputOracle.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.l2OutputOracleProxy),
            address(config.implementationAddressConfig.l2OutputOracleImpl),
            abi.encodeCall(
                L2OutputOracle.initialize,
                (
                    config.l2OutputOracleConfig.l2OutputOracleGenesisL2Output,
                    config.l2OutputOracleConfig.l2OutputOracleProposer,
                    config.l2OutputOracleConfig.l2OutputOracleOwner
                )
            )
        );

        // Upgrade and initialize the OptimismPortal.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.optimismPortalProxy),
            address(config.implementationAddressConfig.optimismPortalImpl),
            abi.encodeCall(OptimismPortal.initialize, ())
        );

        // Upgrade and initialize the L1CrossDomainMessenger.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.l1CrossDomainMessengerProxy),
            address(config.implementationAddressConfig.l1CrossDomainMessengerImpl),
            abi.encodeCall(L1CrossDomainMessenger.initialize, ())
        );

        // Upgrade the L1StandardBridge (no initializer).
        config.globalConfig.proxyAdmin.upgrade(
            payable(config.proxyAddressConfig.l1StandardBridgeProxy),
            address(config.implementationAddressConfig.l1StandardBridgeImpl)
        );

        // Upgrade the OptimismMintableERC20Factory (no initializer).
        config.globalConfig.proxyAdmin.upgrade(
            payable(config.proxyAddressConfig.optimismMintableERC20FactoryProxy),
            address(config.implementationAddressConfig.optimismMintableERC20FactoryImpl)
        );

        // Upgrade the L1ERC721Bridge (no initializer).
        config.globalConfig.proxyAdmin.upgrade(
            payable(config.proxyAddressConfig.l1ERC721BridgeProxy),
            address(config.implementationAddressConfig.l1ERC721BridgeImpl)
        );
    }

    function step2() external onlyOwner {
        // Transfer ownership of the ProxyAdmin to the final owner.
        config.globalConfig.proxyAdmin.setOwner(config.globalConfig.finalOwner);
    }
}
