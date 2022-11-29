// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2OutputOracle } from "../L1/L2OutputOracle.sol";
import { OptimismPortal } from "../L1/OptimismPortal.sol";
import { L1CrossDomainMessenger } from "../L1/L1CrossDomainMessenger.sol";
import { L1ChugSplashProxy } from "../legacy/L1ChugSplashProxy.sol";
import { ProxyAdmin } from "../universal/ProxyAdmin.sol";
import { PortalSender } from "./PortalSender.sol";
import { SystemConfig } from "../L1/SystemConfig.sol";
import { BaseSystemDictator } from "./BaseSystemDictator.sol";

/**
 * @title MigrationSystemDictator
 * @notice The MigrationSystemDictator is responsible for coordinating the migration and
 *         initialization of an existing deployment of the Optimism System. We expect that all
 *         proxies and implementations already be deployed before this contract is used.
 */
contract MigrationSystemDictator is BaseSystemDictator {
    /**
     * @notice Step after which exit 1 can no longer be used.
     */
    uint8 public constant EXIT_1_NO_RETURN_STEP = 3;

    /**
     * @notice Step where proxy ownership is transferred.
     */
    uint8 public constant PROXY_TRANSFER_STEP = 4;

    /**
     * @notice Whether or not the deployment is finalized.
     */
    bool public finalized;

    /**
     * @notice Address of the old L1CrossDomainMessenger implementation.
     */
    address public oldL1CrossDomainMessenger;

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
     * @notice Pauses the system by shutting down the L1CrossDomainMessenger and setting the
     *         deposit halt flag to tell the Sequencer's DTL to stop accepting deposits.
     */
    function step2() external onlyOwner step(2) {
        // Store the address of the old L1CrossDomainMessenger implementation. We will need this
        // address in the case that we have to exit early.
        oldL1CrossDomainMessenger = config.globalConfig.addressManager.getAddress(
            "OVM_L1CrossDomainMessenger"
        );

        // Temporarily brick the L1CrossDomainMessenger by setting its implementation address to
        // address(0) which will cause the ResolvedDelegateProxy to revert. Better than pausing
        // the L1CrossDomainMessenger via pause() because it can be easily reverted.
        config.globalConfig.addressManager.setAddress("OVM_L1CrossDomainMessenger", address(0));

        // Set the DTL shutoff block, which will tell the DTL to stop syncing new deposits from the
        // CanonicalTransactionChain. We do this by setting an address in the AddressManager
        // because the DTL already has a reference to the AddressManager and this way we don't also
        // need to give it a reference to the SystemDictator.
        config.globalConfig.addressManager.setAddress(
            "DTL_SHUTOFF_BLOCK",
            address(uint160(block.number))
        );
    }

    /**
     * @notice Removes deprecated addresses from the AddressManager.
     */
    function step3() external onlyOwner step(EXIT_1_NO_RETURN_STEP) {
        // Remove all deprecated addresses from the AddressManager
        string[17] memory deprecated = [
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
            "OVM_L1MultiMessageRelayer",
            "BondManager"
        ];

        for (uint256 i = 0; i < deprecated.length; i++) {
            config.globalConfig.addressManager.setAddress(deprecated[i], address(0));
        }
    }

    /**
     * @notice Transfers system ownership to the ProxyAdmin.
     */
    function step4() external onlyOwner step(PROXY_TRANSFER_STEP) {
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
    function step5() external onlyOwner step(5) {
        // Dynamic config must be set before we can initialize the L2OutputOracle.
        require(
            dynamicConfigSet,
            "MigrationSystemDictator: dynamic oracle config is not yet initialized"
        );

        // Upgrade and initialize the L2OutputOracle.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.l2OutputOracleProxy),
            address(config.implementationAddressConfig.l2OutputOracleImpl),
            abi.encodeCall(
                L2OutputOracle.initialize,
                (
                    l2OutputOracleDynamicConfig.l2OutputOracleStartingBlockNumber,
                    l2OutputOracleDynamicConfig.l2OutputOracleStartingTimestamp
                )
            )
        );

        // Upgrade and initialize the OptimismPortal.
        config.globalConfig.proxyAdmin.upgradeAndCall(
            payable(config.proxyAddressConfig.optimismPortalProxy),
            address(config.implementationAddressConfig.optimismPortalImpl),
            abi.encodeCall(OptimismPortal.initialize, ())
        );

        // Upgrade the L1CrossDomainMessenger.
        config.globalConfig.proxyAdmin.upgrade(
            payable(config.proxyAddressConfig.l1CrossDomainMessengerProxy),
            address(config.implementationAddressConfig.l1CrossDomainMessengerImpl)
        );

        // Try to initialize the L1CrossDomainMessenger, only fail if it's already been initialized.
        try
            L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy)
                .initialize(address(this))
        {
            // L1CrossDomainMessenger is the one annoying edge case difference between existing
            // networks and fresh networks because in existing networks it'll already be
            // initialized but in fresh networks it won't be. Try/catch is the easiest and most
            // consistent way to handle this because initialized() is not exposed publicly.
        } catch Error(string memory reason) {
            require(
                keccak256(abi.encodePacked(reason)) ==
                    keccak256("Initializable: contract is already initialized"),
                string.concat(
                    "MigrationSystemDictator: unexpected error initializing L1XDM: ",
                    reason
                )
            );
        } catch {
            revert("MigrationSystemDictator: unexpected error initializing L1XDM (no reason)");
        }

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

        // Pause the L1CrossDomainMessenger, chance to check that everything is OK.
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy).pause();
    }

    /**
     * @notice Unpauses the system at which point the system should be fully operational.
     */
    function step6() external onlyOwner step(6) {
        // Unpause the L1CrossDomainMessenger.
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy).unpause();
    }

    /**
     * @notice Tranfers admin ownership to the final owner.
     */
    function finalize() external onlyOwner {
        // Transfer ownership of the L1CrossDomainMessenger to the final owner.
        L1CrossDomainMessenger(config.proxyAddressConfig.l1CrossDomainMessengerProxy)
            .transferOwnership(config.globalConfig.finalOwner);

        // Transfer ownership of the ProxyAdmin to the final owner.
        config.globalConfig.proxyAdmin.transferOwnership(config.globalConfig.finalOwner);

        // Optionally also transfer AddressManager and L1StandardBridge if we still own it. Might
        // happen if we're exiting early.
        if (currentStep <= PROXY_TRANSFER_STEP) {
            // Transfer ownership of the AddressManager to the final owner.
            config.globalConfig.addressManager.transferOwnership(
                address(config.globalConfig.finalOwner)
            );

            // Transfer ownership of the L1StandardBridge to the final owner.
            L1ChugSplashProxy(payable(config.proxyAddressConfig.l1StandardBridgeProxy)).setOwner(
                address(config.globalConfig.finalOwner)
            );
        }

        finalized = true;
    }

    /**
     * @notice First exit point, can only be called before step 3 is executed.
     */
    function exit1() external onlyOwner {
        require(
            currentStep == EXIT_1_NO_RETURN_STEP,
            "MigrationSystemDictator: can only exit1 before step 3 is executed"
        );

        // Reset the L1CrossDomainMessenger to the old implementation.
        config.globalConfig.addressManager.setAddress(
            "OVM_L1CrossDomainMessenger",
            oldL1CrossDomainMessenger
        );

        // Unset the DTL shutoff block which will allow the DTL to sync again.
        config.globalConfig.addressManager.setAddress("DTL_SHUTOFF_BLOCK", address(0));
    }
}
