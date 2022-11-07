// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { PortalSender } from "./PortalSender.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";
import { DeployConfig } from "./DeployConfig.sol";
import { BaseSystemDictator } from "./BaseSystemDictator.sol";

/**
 * @title MigrationSystemDictator
 * @notice The MigrationSystemDictator is responsible for coordinating the migration and
 *         initialization of an existing deployment of the Optimism System. We expect that all
 *         proxies and implementations already be deployed before this contract is used.
 */
contract MigrationSystemDictator is BaseSystemDictator {
    /**
     * @param _config System configuration.
     */
    constructor(DeployConfig memory _config) BaseSystemDictator(_config) {}

    /**
     * @notice Configures the ProxyAdmin contract.
     */
    function step1() external onlyOwner step(1) {
        // Set the AddressManager in the ProxyAdmin.
        config.globalConfig.proxyAdmin.setAddressManager(config.globalConfig.addressManager);

        // Set the L1CrossDomainMessenger to the RESOLVED proxy type.
        config.globalConfig.proxyAdmin.setProxyType(
            config.proxyAddressConfig.l1CrossDomainMessengerProxy,
            ProxyAdmin.ProxyType.RESOLVED
        );

        // Set the implementation name for the L1CrossDomainMessenger.
        config.globalConfig.proxyAdmin.setImplementationName(
            config.proxyAddressConfig.l1CrossDomainMessengerProxy,
            "OVM_L1CrossDomainMessenger"
        );

        // Set the L1StandardBridge to the CHUGSPLASH proxy type.
        config.globalConfig.proxyAdmin.setProxyType(
            config.proxyAddressConfig.l1StandardBridgeProxy,
            ProxyAdmin.ProxyType.CHUGSPLASH
        );
    }

    /**
     * @notice Pauses the system by shutting down the L1CrossDomainMessenger and clearing many
     *         addresses inside the AddressManager.
     */
    function step2() external onlyOwner step(2) {
        // Pause the L1CrossDomainMessenger
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy).pause();

        // Remove all dead addresses from the AddressManager
        string[18] memory deads = [
            "Proxy__OVM_L1CrossDomainMessenger",
            "Proxy__OVM_L1StandardBridge",
            "OVM_CanonicalTransactionChain",
            "OVM_L2CrossDomainMessenger",
            "OVM_DecompressionPrecompileAddress",
            "OVM_Sequencer",
            "OVM_Proposer",
            "OVM_ChainStorageContainer-CTC-batches",
            "OVM_ChainStorageContainer-CTC-queue",
            "OVM_CanonicalTransactionChain",
            "OVM_StateCommitmentChain",
            "OVM_BondManager",
            "OVM_ExecutionManager",
            "OVM_FraudVerifier",
            "OVM_StateManagerFactory",
            "OVM_StateTransitionerFactory",
            "OVM_SafetyChecker",
            "OVM_L1MultiMessageRelayer"
        ];

        for (uint256 i = 0; i < deads.length; i++) {
            config.globalConfig.addressManager.setAddress(deads[i], address(0));
        }
    }

    /**
     * @notice Transfers system ownership to the ProxyAdmin.
     */
    function step3() external onlyOwner step(3) {
        // Transfer ownership of the AddressManager to the ProxyAdmin.
        config.globalConfig.addressManager.transferOwnership(
            address(config.globalConfig.proxyAdmin)
        );

        // Transfer ownership of the L1StandardBridge to the ProxyAdmin.
        L1ChugSplashProxy(payable(config.proxyAddressConfig.l1StandardBridgeProxy)).setOwner(
            address(config.globalConfig.proxyAdmin)
        );
    }

    /**
     * @notice Upgrades and initializes proxy contracts.
     */
    function step4() external onlyOwner step(4) {
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

        // Upgrade the L1CrossDomainMessenger. No initializer because this is
        // already initialized.
        config.globalConfig.proxyAdmin.upgrade(
            payable(config.proxyAddressConfig.l1CrossDomainMessengerProxy),
            address(config.implementationAddressConfig.l1CrossDomainMessengerImpl)
        );

        // Transfer ETH from the L1StandardBridge to the OptimismPortal.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.l1StandardBridgeProxy),
            address(config.implementationAddressConfig.portalSenderImpl),
            abi.encodeCall(PortalSender.donate, ())
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
     * @notice Unpauses the system at which point the system should be fully operational.
     */
    function step5() external onlyOwner step(5) {
        // Unpause the L1CrossDomainMessenger.
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy).unpause();
    }

    /**
     * @notice Tranfers admin ownership to the final owner.
     */
    function step6() external onlyOwner step(6) {
        // Transfer ownership of the L1CrossDomainMessenger to the final owner.
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy)
            .transferOwnership(config.globalConfig.finalOwner);

        // Transfer ownership of the ProxyAdmin to the final owner.
        config.globalConfig.proxyAdmin.setOwner(config.globalConfig.finalOwner);
    }
}
