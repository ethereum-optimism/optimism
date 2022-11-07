// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { DeployConfig } from "./DeployConfig.sol";
import { BaseSystemDictator } from "./BaseSystemDictator.sol";

/**
 * @title FreshSystemDictator
 * @notice The FreshSystemDictator is responsible for coordinating initialization of a fresh
 *         deployment of the Optimism system. We expect that all proxies and implementations
 *         already be deployed before this contract is used.
 */
contract FreshSystemDictator is BaseSystemDictator {
    /**
     * @param _config System configuration.
     */
    constructor(DeployConfig memory _config) BaseSystemDictator(_config) {}

    /**
     * @notice Upgrades and initializes proxy contracts.
     */
    function step1() external onlyOwner step(1) {
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
            abi.encodeCall(L1CrossDomainMessenger.initialize, (config.globalConfig.finalOwner))
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

        // Upgrade and initialize the SystemConfig.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.systemConfigProxy),
            address(config.implementationAddressConfig.systemConfigImpl),
            abi.encodeCall(
                SystemConfig.initialize,
                (
                    config.systemConfigConfig.owner,
                    config.systemConfigConfig.overhead,
                    config.systemConfigConfig.scalar,
                    config.systemConfigConfig.batcherHash,
                    config.systemConfigConfig.gasLimit
                )
            )
        );
    }

    /**
     * @notice Transfers ownership to final owner.
     */
    function step2() external onlyOwner step(2) {
        // Transfer ownership of the ProxyAdmin to the final owner.
        config.globalConfig.proxyAdmin.setOwner(config.globalConfig.finalOwner);
    }
}
