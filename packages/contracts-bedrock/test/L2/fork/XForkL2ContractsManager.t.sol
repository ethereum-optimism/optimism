// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { XForkL2ContractsManager } from "src/L2/XForkL2ContractsManager.sol";
import { XForkL2CMTypes } from "src/libraries/XForkL2CMTypes.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";

import { StorageSetter } from "src/universal/StorageSetter.sol";
import { WETH } from "src/L2/WETH.sol";
import { GasPriceOracle } from "src/L2/GasPriceOracle.sol";
import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { OptimismMintableERC20Factory } from "src/universal/OptimismMintableERC20Factory.sol";
import { L2ERC721Bridge } from "src/L2/L2ERC721Bridge.sol";
import { L1Block } from "src/L2/L1Block.sol";
import { L1BlockCGT } from "src/L2/L1BlockCGT.sol";
import { L2ToL1MessagePasser } from "src/L2/L2ToL1MessagePasser.sol";
import { L2ToL1MessagePasserCGT } from "src/L2/L2ToL1MessagePasserCGT.sol";
import { OptimismMintableERC721Factory } from "src/L2/OptimismMintableERC721Factory.sol";
import { ProxyAdmin } from "src/universal/ProxyAdmin.sol";
import { GovernanceToken } from "src/governance/GovernanceToken.sol";
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";
import { OptimismSuperchainERC20Beacon } from "src/L2/OptimismSuperchainERC20Beacon.sol";
import { NativeAssetLiquidity } from "src/L2/NativeAssetLiquidity.sol";
import { LiquidityController } from "src/L2/LiquidityController.sol";

/// @title XForkL2ContractsManager_Harness
/// @notice Harness contract that exposes internal functions for testing.
contract XForkL2ContractsManager_Harness is XForkL2ContractsManager {
    constructor(XForkL2CMTypes.Implementations memory _implementations) XForkL2ContractsManager(_implementations) { }

    /// @notice Returns the full configuration for the L2 predeploys.
    function fullConfig() external view returns (XForkL2CMTypes.FullConfig memory) {
        return _fullConfig();
    }

    /// @notice Returns the target implementations for the L2 predeploys.
    function implementations() external view returns (XForkL2CMTypes.Implementations memory) {
        return XForkL2CMTypes.Implementations({
            storageSetterImpl: STORAGE_SETTER_IMPL,
            l2CrossDomainMessengerImpl: L2_CROSS_DOMAIN_MESSENGER_IMPL,
            gasPriceOracleImpl: GAS_PRICE_ORACLE_IMPL,
            l2StandardBridgeImpl: L2_STANDARD_BRIDGE_IMPL,
            sequencerFeeWalletImpl: SEQUENCER_FEE_WALLET_IMPL,
            optimismMintableERC20FactoryImpl: OPTIMISM_MINTABLE_ERC20_FACTORY_IMPL,
            l2ERC721BridgeImpl: L2_ERC721_BRIDGE_IMPL,
            l1BlockAttributesImpl: L1_BLOCK_ATTRIBUTES_IMPL,
            l1BlockAttributesCGTImpl: L1_BLOCK_ATTRIBUTES_CGT_IMPL,
            l2ToL1MessagePasserImpl: L2_TO_L1_MESSAGE_PASSER_IMPL,
            l2ToL1MessagePasserCGTImpl: L2_TO_L1_MESSAGE_PASSER_CGT_IMPL,
            optimismMintableERC721FactoryImpl: OPTIMISM_MINTABLE_ERC721_FACTORY_IMPL,
            proxyAdminImpl: PROXY_ADMIN_IMPL,
            baseFeeVaultImpl: BASE_FEE_VAULT_IMPL,
            l1FeeVaultImpl: L1_FEE_VAULT_IMPL,
            operatorFeeVaultImpl: OPERATOR_FEE_VAULT_IMPL,
            schemaRegistryImpl: SCHEMA_REGISTRY_IMPL,
            easImpl: EAS_IMPL,
            crossL2InboxImpl: CROSS_L2_INBOX_IMPL,
            l2ToL2CrossDomainMessengerImpl: L2_TO_L2_CROSS_DOMAIN_MESSENGER_IMPL,
            superchainETHBridgeImpl: SUPERCHAIN_ETH_BRIDGE_IMPL,
            ethLiquidityImpl: ETH_LIQUIDITY_IMPL,
            optimismSuperchainERC20FactoryImpl: OPTIMISM_SUPERCHAIN_ERC20_FACTORY_IMPL,
            optimismSuperchainERC20BeaconImpl: OPTIMISM_SUPERCHAIN_ERC20_BEACON_IMPL,
            superchainTokenBridgeImpl: SUPERCHAIN_TOKEN_BRIDGE_IMPL,
            nativeAssetLiquidityImpl: NATIVE_ASSET_LIQUIDITY_IMPL,
            liquidityControllerImpl: LIQUIDITY_CONTROLLER_IMPL,
            feeSplitterImpl: FEE_SPLITTER_IMPL
        });
    }
}

/// @title XForkL2ContractsManager_Test
/// @notice Test contract for the XForkL2ContractsManager contract, testing the upgrade path for this fork.
contract XForkL2ContractsManager_Test is CommonTest {
    XForkL2ContractsManager_Harness internal l2cm;
    XForkL2CMTypes.Implementations internal implementations;

    /// @notice Struct to capture the post-upgrade state for comparison.
    struct PostUpgradeState {
        // Implementation addresses
        address wethImpl;
        address gasPriceOracleImpl;
        address l2CrossDomainMessengerImpl;
        address l2StandardBridgeImpl;
        address sequencerFeeWalletImpl;
        address optimismMintableERC20FactoryImpl;
        address l2ERC721BridgeImpl;
        address l1BlockAttributesImpl;
        address l2ToL1MessagePasserImpl;
        address optimismMintableERC721FactoryImpl;
        address proxyAdminImpl;
        address baseFeeVaultImpl;
        address l1FeeVaultImpl;
        address operatorFeeVaultImpl;
        address schemaRegistryImpl;
        address easImpl;
        address governanceTokenImpl;
        address crossL2InboxImpl;
        address l2ToL2CrossDomainMessengerImpl;
        address superchainETHBridgeImpl;
        address ethLiquidityImpl;
        address optimismSuperchainERC20FactoryImpl;
        address optimismSuperchainERC20BeaconImpl;
        address superchainTokenBridgeImpl;
        address nativeAssetLiquidityImpl;
        address liquidityControllerImpl;
        address feeSplitterImpl;
        // Config values, take advantage of the harness to capture the config values
        XForkL2CMTypes.FullConfig config;
    }

    function setUp() public override {
        super.setUp();
        _loadImplementations();
        _deployL2CM();
    }

    /// @notice Deploys the target implementations for the predeploys.
    function _loadImplementations() internal {
        // Deploy a fresh StorageSetter for the upgrade process
        implementations.storageSetterImpl = address(new StorageSetter());

        implementations.gasPriceOracleImpl = address(new GasPriceOracle());
        implementations.l2CrossDomainMessengerImpl = address(new L2CrossDomainMessenger());
        implementations.l2StandardBridgeImpl = address(new L2StandardBridge());
        implementations.optimismMintableERC20FactoryImpl = address(new OptimismMintableERC20Factory());
        implementations.l2ERC721BridgeImpl = address(new L2ERC721Bridge());
        implementations.l1BlockAttributesImpl = address(new L1Block());
        implementations.l1BlockAttributesCGTImpl = address(new L1BlockCGT());
        implementations.l2ToL1MessagePasserImpl = address(new L2ToL1MessagePasser());
        implementations.l2ToL1MessagePasserCGTImpl = address(new L2ToL1MessagePasserCGT());
        implementations.optimismMintableERC721FactoryImpl = address(new OptimismMintableERC721Factory(address(0), 0));
        implementations.proxyAdminImpl = address(new ProxyAdmin(address(0)));
        implementations.superchainETHBridgeImpl = address(new SuperchainETHBridge());
        implementations.ethLiquidityImpl = address(new ETHLiquidity());
        implementations.optimismSuperchainERC20BeaconImpl = address(new OptimismSuperchainERC20Beacon());
        implementations.nativeAssetLiquidityImpl = address(new NativeAssetLiquidity());
        implementations.liquidityControllerImpl = address(new LiquidityController());

        // Deploy 0.8.19 contracts using deployCode()
        implementations.schemaRegistryImpl = deployCode("src/vendor/eas/SchemaRegistry.sol:SchemaRegistry");
        implementations.easImpl = deployCode("src/vendor/eas/EAS.sol:EAS");

        // Deploy 0.8.25 contracts using deployCode()
        implementations.baseFeeVaultImpl = deployCode("src/L2/BaseFeeVault.sol:BaseFeeVault");
        implementations.l1FeeVaultImpl = deployCode("src/L2/L1FeeVault.sol:L1FeeVault");
        implementations.operatorFeeVaultImpl = deployCode("src/L2/OperatorFeeVault.sol:OperatorFeeVault");
        implementations.sequencerFeeWalletImpl = deployCode("src/L2/SequencerFeeVault.sol:SequencerFeeVault");
        implementations.crossL2InboxImpl = deployCode("src/L2/CrossL2Inbox.sol:CrossL2Inbox");
        implementations.l2ToL2CrossDomainMessengerImpl =
            deployCode("src/L2/L2ToL2CrossDomainMessenger.sol:L2ToL2CrossDomainMessenger");
        implementations.optimismSuperchainERC20FactoryImpl =
            deployCode("src/L2/OptimismSuperchainERC20Factory.sol:OptimismSuperchainERC20Factory");
        implementations.superchainTokenBridgeImpl = deployCode("src/L2/SuperchainTokenBridge.sol:SuperchainTokenBridge");
        implementations.feeSplitterImpl = deployCode("src/L2/FeeSplitter.sol:FeeSplitter");
    }

    /// @notice Deploys the XForkL2ContractsManager with the loaded implementations.
    function _deployL2CM() internal {
        l2cm = new XForkL2ContractsManager_Harness(implementations);
        vm.label(address(l2cm), "XForkL2ContractsManager");
    }

    /// @notice Executes the upgrade via DELEGATECALL from the L2ProxyAdmin context.
    function _executeUpgrade() internal {
        // The L2CM must be called via DELEGATECALL from the ProxyAdmin.
        // We simulate this by pranking as the ProxyAdmin and using delegatecall.
        address proxyAdmin = Predeploys.PROXY_ADMIN;
        prankDelegateCall(proxyAdmin);
        (bool success,) = address(l2cm).delegatecall(abi.encodeCall(XForkL2ContractsManager.upgrade, ()));
        require(success, "Upgrade failed");
    }

    /// @notice Captures the current post-upgrade state of all predeploys.
    /// @return state_ The captured state.
    function _capturePostUpgradeState() internal view returns (PostUpgradeState memory state_) {
        // Capture implementation addresses
        state_.wethImpl = EIP1967Helper.getImplementation(Predeploys.WETH);
        state_.gasPriceOracleImpl = EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE);
        state_.l2CrossDomainMessengerImpl = EIP1967Helper.getImplementation(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        state_.l2StandardBridgeImpl = EIP1967Helper.getImplementation(Predeploys.L2_STANDARD_BRIDGE);
        state_.sequencerFeeWalletImpl = EIP1967Helper.getImplementation(Predeploys.SEQUENCER_FEE_WALLET);
        state_.optimismMintableERC20FactoryImpl =
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
        state_.l2ERC721BridgeImpl = EIP1967Helper.getImplementation(Predeploys.L2_ERC721_BRIDGE);
        state_.l1BlockAttributesImpl = EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES);
        state_.l2ToL1MessagePasserImpl = EIP1967Helper.getImplementation(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        state_.optimismMintableERC721FactoryImpl =
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY);
        state_.proxyAdminImpl = EIP1967Helper.getImplementation(Predeploys.PROXY_ADMIN);
        state_.baseFeeVaultImpl = EIP1967Helper.getImplementation(Predeploys.BASE_FEE_VAULT);
        state_.l1FeeVaultImpl = EIP1967Helper.getImplementation(Predeploys.L1_FEE_VAULT);
        state_.operatorFeeVaultImpl = EIP1967Helper.getImplementation(Predeploys.OPERATOR_FEE_VAULT);
        state_.schemaRegistryImpl = EIP1967Helper.getImplementation(Predeploys.SCHEMA_REGISTRY);
        state_.easImpl = EIP1967Helper.getImplementation(Predeploys.EAS);
        state_.governanceTokenImpl = EIP1967Helper.getImplementation(Predeploys.GOVERNANCE_TOKEN);
        state_.crossL2InboxImpl = EIP1967Helper.getImplementation(Predeploys.CROSS_L2_INBOX);
        state_.l2ToL2CrossDomainMessengerImpl =
            EIP1967Helper.getImplementation(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        state_.superchainETHBridgeImpl = EIP1967Helper.getImplementation(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        state_.ethLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.ETH_LIQUIDITY);
        state_.optimismSuperchainERC20FactoryImpl =
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY);
        state_.optimismSuperchainERC20BeaconImpl =
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        state_.superchainTokenBridgeImpl = EIP1967Helper.getImplementation(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
        state_.nativeAssetLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.NATIVE_ASSET_LIQUIDITY);
        state_.liquidityControllerImpl = EIP1967Helper.getImplementation(Predeploys.LIQUIDITY_CONTROLLER);
        state_.feeSplitterImpl = EIP1967Helper.getImplementation(Predeploys.FEE_SPLITTER);

        // Capture config values using the harness
        state_.config = l2cm.fullConfig();
    }

    /// @notice Asserts that two post-upgrade states are identical.
    /// @param _state1 The first state.
    /// @param _state2 The second state.
    function _assertStatesEqual(PostUpgradeState memory _state1, PostUpgradeState memory _state2) internal pure {
        // Assert implementation addresses are equal
        assertEq(_state1.wethImpl, _state2.wethImpl, "WETH impl mismatch");
        assertEq(_state1.gasPriceOracleImpl, _state2.gasPriceOracleImpl, "GasPriceOracle impl mismatch");
        assertEq(
            _state1.l2CrossDomainMessengerImpl,
            _state2.l2CrossDomainMessengerImpl,
            "L2CrossDomainMessenger impl mismatch"
        );
        assertEq(_state1.l2StandardBridgeImpl, _state2.l2StandardBridgeImpl, "L2StandardBridge impl mismatch");
        assertEq(_state1.sequencerFeeWalletImpl, _state2.sequencerFeeWalletImpl, "SequencerFeeWallet impl mismatch");
        assertEq(
            _state1.optimismMintableERC20FactoryImpl,
            _state2.optimismMintableERC20FactoryImpl,
            "OptimismMintableERC20Factory impl mismatch"
        );
        assertEq(_state1.l2ERC721BridgeImpl, _state2.l2ERC721BridgeImpl, "L2ERC721Bridge impl mismatch");
        assertEq(_state1.l1BlockAttributesImpl, _state2.l1BlockAttributesImpl, "L1BlockAttributes impl mismatch");
        assertEq(_state1.l2ToL1MessagePasserImpl, _state2.l2ToL1MessagePasserImpl, "L2ToL1MessagePasser impl mismatch");
        assertEq(
            _state1.optimismMintableERC721FactoryImpl,
            _state2.optimismMintableERC721FactoryImpl,
            "OptimismMintableERC721Factory impl mismatch"
        );
        assertEq(_state1.proxyAdminImpl, _state2.proxyAdminImpl, "ProxyAdmin impl mismatch");
        assertEq(_state1.baseFeeVaultImpl, _state2.baseFeeVaultImpl, "BaseFeeVault impl mismatch");
        assertEq(_state1.l1FeeVaultImpl, _state2.l1FeeVaultImpl, "L1FeeVault impl mismatch");
        assertEq(_state1.operatorFeeVaultImpl, _state2.operatorFeeVaultImpl, "OperatorFeeVault impl mismatch");
        assertEq(_state1.schemaRegistryImpl, _state2.schemaRegistryImpl, "SchemaRegistry impl mismatch");
        assertEq(_state1.easImpl, _state2.easImpl, "EAS impl mismatch");
        assertEq(_state1.governanceTokenImpl, _state2.governanceTokenImpl, "GovernanceToken impl mismatch");
        assertEq(_state1.crossL2InboxImpl, _state2.crossL2InboxImpl, "CrossL2Inbox impl mismatch");
        assertEq(
            _state1.l2ToL2CrossDomainMessengerImpl,
            _state2.l2ToL2CrossDomainMessengerImpl,
            "L2ToL2CrossDomainMessenger impl mismatch"
        );
        assertEq(_state1.superchainETHBridgeImpl, _state2.superchainETHBridgeImpl, "SuperchainETHBridge impl mismatch");
        assertEq(_state1.ethLiquidityImpl, _state2.ethLiquidityImpl, "ETHLiquidity impl mismatch");
        assertEq(
            _state1.optimismSuperchainERC20FactoryImpl,
            _state2.optimismSuperchainERC20FactoryImpl,
            "OptimismSuperchainERC20Factory impl mismatch"
        );
        assertEq(
            _state1.optimismSuperchainERC20BeaconImpl,
            _state2.optimismSuperchainERC20BeaconImpl,
            "OptimismSuperchainERC20Beacon impl mismatch"
        );
        assertEq(
            _state1.superchainTokenBridgeImpl, _state2.superchainTokenBridgeImpl, "SuperchainTokenBridge impl mismatch"
        );
        assertEq(
            _state1.nativeAssetLiquidityImpl, _state2.nativeAssetLiquidityImpl, "NativeAssetLiquidity impl mismatch"
        );
        assertEq(_state1.liquidityControllerImpl, _state2.liquidityControllerImpl, "LiquidityController impl mismatch");
        assertEq(_state1.feeSplitterImpl, _state2.feeSplitterImpl, "FeeSplitter impl mismatch");

        // Assert config values are equal
        assertEq(
            _state1.config.crossDomainMessenger.otherMessenger,
            _state2.config.crossDomainMessenger.otherMessenger,
            "CrossDomainMessenger config mismatch"
        );
        assertEq(
            _state1.config.standardBridge.otherBridge,
            _state2.config.standardBridge.otherBridge,
            "StandardBridge config mismatch"
        );
        assertEq(
            _state1.config.erc721Bridge.otherBridge,
            _state2.config.erc721Bridge.otherBridge,
            "ERC721Bridge config mismatch"
        );
        assertEq(
            _state1.config.mintableERC20Factory.bridge,
            _state2.config.mintableERC20Factory.bridge,
            "MintableERC20Factory config mismatch"
        );
        assertEq(
            _state1.config.sequencerFeeVault.recipient,
            _state2.config.sequencerFeeVault.recipient,
            "SequencerFeeVault recipient mismatch"
        );
        assertEq(
            _state1.config.baseFeeVault.recipient,
            _state2.config.baseFeeVault.recipient,
            "BaseFeeVault recipient mismatch"
        );
        assertEq(
            _state1.config.l1FeeVault.recipient, _state2.config.l1FeeVault.recipient, "L1FeeVault recipient mismatch"
        );
        assertEq(
            _state1.config.operatorFeeVault.recipient,
            _state2.config.operatorFeeVault.recipient,
            "OperatorFeeVault recipient mismatch"
        );
        assertEq(
            _state1.config.liquidityController.owner,
            _state2.config.liquidityController.owner,
            "LiquidityController owner mismatch"
        );
        assertEq(
            _state1.config.feeSplitter.sharesCalculator,
            _state2.config.feeSplitter.sharesCalculator,
            "FeeSplitter sharesCalculator mismatch"
        );
    }

    /// @notice Tests that the upgrade produces identical state when called twice with the same pre-state.
    function test_upgrade_producesSameState_whenCalledTwiceWithSamePreState() public {
        // Save the pre-upgrade state
        uint256 snapshotId = vm.snapshotState();

        // Execute the first upgrade
        _executeUpgrade();

        // Capture the post-upgrade state after first execution
        PostUpgradeState memory stateAfterFirstUpgrade = _capturePostUpgradeState();

        // Revert to the pre-upgrade state
        vm.revertToState(snapshotId);

        // Execute the second upgrade (L2CM and impls are preserved from the snapshot)
        _executeUpgrade();

        // Capture the post-upgrade state after second execution
        PostUpgradeState memory stateAfterSecondUpgrade = _capturePostUpgradeState();

        // Assert both states are identical
        _assertStatesEqual(stateAfterFirstUpgrade, stateAfterSecondUpgrade);
    }
}
