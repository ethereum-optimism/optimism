// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Contracts
import { L2ContractsManager } from "src/L2/L2ContractsManager.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { IProxy } from "interfaces/universal/IProxy.sol";

/// @title XForkContractsManager
/// @notice The XForkContractsManager is responsible for orquestrating the upgrades of the L2 contracts during xFork
/// hardforks.
contract XForkContractsManager is L2ContractsManager {
    /// @notice Configuration for all L2 predeploy implementation addresses.
    /// @param legacyMessagePasserImplementation Implementation for LegacyMessagePasser.
    /// @param deployerWhitelistImplementation Implementation for DeployerWhitelist.
    /// @param l2CrossDomainMessengerImplementation Implementation for L2CrossDomainMessenger.
    /// @param gasPriceOracleImplementation Implementation for GasPriceOracle.
    /// @param l2StandardBridgeImplementation Implementation for L2StandardBridge.
    /// @param sequencerFeeWalletImplementation Implementation for SequencerFeeWallet.
    /// @param optimismMintableERC20FactoryImplementation Implementation for OptimismMintableERC20Factory.
    /// @param l1BlockNumberImplementation Implementation for L1BlockNumber.
    /// @param l2ERC721BridgeImplementation Implementation for L2ERC721Bridge.
    /// @param l1BlockAttributesImplementation Implementation for L1Block.
    /// @param l2ToL1MessagePasserImplementation Implementation for L2ToL1MessagePasser.
    /// @param optimismMintableERC721FactoryImplementation Implementation for OptimismMintableERC721Factory.
    /// @param baseFeeVaultImplementation Implementation for BaseFeeVault.
    /// @param l1FeeVaultImplementation Implementation for L1FeeVault.
    /// @param operatorFeeVaultImplementation Implementation for OperatorFeeVault.
    /// @param schemaRegistryImplementation Implementation for SchemaRegistry.
    /// @param easImplementation Implementation for EAS.
    struct Input {
        address legacyMessagePasserImplementation;
        address deployerWhitelistImplementation;
        address l2CrossDomainMessengerImplementation;
        address gasPriceOracleImplementation;
        address l2StandardBridgeImplementation;
        address sequencerFeeWalletImplementation;
        address optimismMintableERC20FactoryImplementation;
        address l1BlockNumberImplementation;
        address l2ERC721BridgeImplementation;
        address l1BlockAttributesImplementation;
        address l2ToL1MessagePasserImplementation;
        address optimismMintableERC721FactoryImplementation;
        address baseFeeVaultImplementation;
        address l1FeeVaultImplementation;
        address operatorFeeVaultImplementation;
        address schemaRegistryImplementation;
        address easImplementation;
    }

    /// @notice Implementation address for LegacyMessagePasser.
    address internal immutable LEGACY_MESSAGE_PASSER_IMPLEMENTATION;

    /// @notice Implementation address for DeployerWhitelist.
    address internal immutable DEPLOYER_WHITELIST_IMPLEMENTATION;

    /// @notice Implementation address for L2CrossDomainMessenger.
    address internal immutable L2_CROSS_DOMAIN_MESSENGER_IMPLEMENTATION;

    /// @notice Implementation address for GasPriceOracle.
    address internal immutable GAS_PRICE_ORACLE_IMPLEMENTATION;

    /// @notice Implementation address for L2StandardBridge.
    address internal immutable L2_STANDARD_BRIDGE_IMPLEMENTATION;

    /// @notice Implementation address for SequencerFeeWallet.
    address internal immutable SEQUENCER_FEE_WALLET_IMPLEMENTATION;

    /// @notice Implementation address for OptimismMintableERC20Factory.
    address internal immutable OPTIMISM_MINTABLE_ERC20_FACTORY_IMPLEMENTATION;

    /// @notice Implementation address for L1BlockNumber.
    address internal immutable L1_BLOCK_NUMBER_IMPLEMENTATION;

    /// @notice Implementation address for L2ERC721Bridge.
    address internal immutable L2_ERC721_BRIDGE_IMPLEMENTATION;

    /// @notice Implementation address for L1BlockAttributes.
    address internal immutable L1_BLOCK_ATTRIBUTES_IMPLEMENTATION;

    /// @notice Implementation address for L2ToL1MessagePasser.
    address internal immutable L2_TO_L1_MESSAGE_PASSER_IMPLEMENTATION;

    /// @notice Implementation address for OptimismMintableERC721Factory.
    address internal immutable OPTIMISM_MINTABLE_ERC721_FACTORY_IMPLEMENTATION;

    /// @notice Implementation address for BaseFeeVault.
    address internal immutable BASE_FEE_VAULT_IMPLEMENTATION;

    /// @notice Implementation address for L1FeeVault.
    address internal immutable L1_FEE_VAULT_IMPLEMENTATION;

    /// @notice Implementation address for OperatorFeeVault.
    address internal immutable OPERATOR_FEE_VAULT_IMPLEMENTATION;

    /// @notice Implementation address for SchemaRegistry.
    address internal immutable SCHEMA_REGISTRY_IMPLEMENTATION;

    /// @notice Implementation address for EAS.
    address internal immutable EAS_IMPLEMENTATION;

    /// @notice Constructs the XForkContractsManager with implementation addresses.
    /// @dev Reverts if any implementation address is zero.
    /// @param _input Configuration containing all implementation addresses.
    constructor(Input memory _input) {
        LEGACY_MESSAGE_PASSER_IMPLEMENTATION = _input.legacyMessagePasserImplementation;
        DEPLOYER_WHITELIST_IMPLEMENTATION = _input.deployerWhitelistImplementation;
        L2_CROSS_DOMAIN_MESSENGER_IMPLEMENTATION = _input.l2CrossDomainMessengerImplementation;
        GAS_PRICE_ORACLE_IMPLEMENTATION = _input.gasPriceOracleImplementation;
        L2_STANDARD_BRIDGE_IMPLEMENTATION = _input.l2StandardBridgeImplementation;
        SEQUENCER_FEE_WALLET_IMPLEMENTATION = _input.sequencerFeeWalletImplementation;
        OPTIMISM_MINTABLE_ERC20_FACTORY_IMPLEMENTATION = _input.optimismMintableERC20FactoryImplementation;
        L1_BLOCK_NUMBER_IMPLEMENTATION = _input.l1BlockNumberImplementation;
        L2_ERC721_BRIDGE_IMPLEMENTATION = _input.l2ERC721BridgeImplementation;
        L1_BLOCK_ATTRIBUTES_IMPLEMENTATION = _input.l1BlockAttributesImplementation;
        L2_TO_L1_MESSAGE_PASSER_IMPLEMENTATION = _input.l2ToL1MessagePasserImplementation;
        OPTIMISM_MINTABLE_ERC721_FACTORY_IMPLEMENTATION = _input.optimismMintableERC721FactoryImplementation;
        BASE_FEE_VAULT_IMPLEMENTATION = _input.baseFeeVaultImplementation;
        L1_FEE_VAULT_IMPLEMENTATION = _input.l1FeeVaultImplementation;
        OPERATOR_FEE_VAULT_IMPLEMENTATION = _input.operatorFeeVaultImplementation;
        SCHEMA_REGISTRY_IMPLEMENTATION = _input.schemaRegistryImplementation;
        EAS_IMPLEMENTATION = _input.easImplementation;
    }

    /// @notice Hook called before execution.
    function _beforeExecution() internal override { }

    /// @notice Hook called after execution.
    function _afterExecution() internal override { }

    /// @notice Performs upgrades for all L2 predeploy contracts.
    function _performUpgrades() internal override {
        IProxy(payable(Predeploys.LEGACY_MESSAGE_PASSER)).upgradeTo(LEGACY_MESSAGE_PASSER_IMPLEMENTATION);
        IProxy(payable(Predeploys.DEPLOYER_WHITELIST)).upgradeTo(DEPLOYER_WHITELIST_IMPLEMENTATION);
        IProxy(payable(Predeploys.L2_CROSS_DOMAIN_MESSENGER)).upgradeTo(L2_CROSS_DOMAIN_MESSENGER_IMPLEMENTATION);
        IProxy(payable(Predeploys.GAS_PRICE_ORACLE)).upgradeTo(GAS_PRICE_ORACLE_IMPLEMENTATION);
        IProxy(payable(Predeploys.L2_STANDARD_BRIDGE)).upgradeTo(L2_STANDARD_BRIDGE_IMPLEMENTATION);
        IProxy(payable(Predeploys.SEQUENCER_FEE_WALLET)).upgradeTo(SEQUENCER_FEE_WALLET_IMPLEMENTATION);
        IProxy(payable(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY)).upgradeTo(
            OPTIMISM_MINTABLE_ERC20_FACTORY_IMPLEMENTATION
        );
        IProxy(payable(Predeploys.L1_BLOCK_NUMBER)).upgradeTo(L1_BLOCK_NUMBER_IMPLEMENTATION);
        IProxy(payable(Predeploys.L2_ERC721_BRIDGE)).upgradeTo(L2_ERC721_BRIDGE_IMPLEMENTATION);
        IProxy(payable(Predeploys.L1_BLOCK_ATTRIBUTES)).upgradeTo(L1_BLOCK_ATTRIBUTES_IMPLEMENTATION);
        IProxy(payable(Predeploys.L2_TO_L1_MESSAGE_PASSER)).upgradeTo(L2_TO_L1_MESSAGE_PASSER_IMPLEMENTATION);
        IProxy(payable(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY)).upgradeTo(
            OPTIMISM_MINTABLE_ERC721_FACTORY_IMPLEMENTATION
        );
        IProxy(payable(Predeploys.BASE_FEE_VAULT)).upgradeTo(BASE_FEE_VAULT_IMPLEMENTATION);
        IProxy(payable(Predeploys.L1_FEE_VAULT)).upgradeTo(L1_FEE_VAULT_IMPLEMENTATION);
        IProxy(payable(Predeploys.OPERATOR_FEE_VAULT)).upgradeTo(OPERATOR_FEE_VAULT_IMPLEMENTATION);
        IProxy(payable(Predeploys.SCHEMA_REGISTRY)).upgradeTo(SCHEMA_REGISTRY_IMPLEMENTATION);
        IProxy(payable(Predeploys.EAS)).upgradeTo(EAS_IMPLEMENTATION);
    }
}
