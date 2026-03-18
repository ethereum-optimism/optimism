// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { L2ContractsManager } from "src/L2/L2ContractsManager.sol";
import { L2ContractsManagerTypes } from "src/libraries/L2ContractsManagerTypes.sol";
import { L2ContractsManagerUtils } from "src/libraries/L2ContractsManagerUtils.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { StorageSetter } from "src/universal/StorageSetter.sol";
import { L2CrossDomainMessenger } from "src/L2/L2CrossDomainMessenger.sol";
import { Types } from "src/libraries/Types.sol";
import { Features } from "src/libraries/Features.sol";
import { Config } from "scripts/libraries/Config.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

// Contracts
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
import { SuperchainETHBridge } from "src/L2/SuperchainETHBridge.sol";
import { ETHLiquidity } from "src/L2/ETHLiquidity.sol";
import { OptimismSuperchainERC20Beacon } from "src/L2/OptimismSuperchainERC20Beacon.sol";
import { NativeAssetLiquidity } from "src/L2/NativeAssetLiquidity.sol";
import { LiquidityController } from "src/L2/LiquidityController.sol";

/// @title L2ContractsManager_FunctionsExposer_Harness
/// @notice Harness contract that exposes internal functions for testing.
contract L2ContractsManager_FunctionsExposer_Harness is L2ContractsManager {
    constructor(L2ContractsManagerTypes.Implementations memory _implementations) L2ContractsManager(_implementations) { }

    /// @notice Returns the full configuration for the L2 predeploys.
    function loadFullConfig() external view returns (L2ContractsManagerTypes.FullConfig memory) {
        return _loadFullConfig();
    }

    /// @notice Returns true if _feature is enabled and false otherwise.
    function isDevFeatureEnabled(bytes32 _feature) external view returns (bool) {
        return _isDevFeatureEnabled(_feature);
    }
}

/// @title L2ContractsManager_Upgrade_Test
/// @notice Test contract for the L2ContractsManager contract, testing the upgrade path.
contract L2ContractsManager_Upgrade_Test is CommonTest {
    L2ContractsManager_FunctionsExposer_Harness internal l2cm;
    L2ContractsManagerTypes.Implementations internal implementations;

    /// @notice Struct to capture the post-upgrade state for comparison.
    struct PostUpgradeState {
        // Implementation addresses
        address gasPriceOracleImpl;
        address l2CrossDomainMessengerImpl;
        address l2StandardBridgeImpl;
        address sequencerFeeWalletImpl;
        address optimismMintableERC20FactoryImpl;
        address l2ERC721BridgeImpl;
        address l1BlockImpl;
        address l1BlockCGTImpl;
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
        address l2DevFeatureFlagsImpl;
        // Config values, take advantage of the harness to capture the config values
        L2ContractsManagerTypes.FullConfig config;
    }

    function setUp() public virtual override {
        super.setUp();
        _loadImplementations();
        _deployL2CM();

        skipIfDevFeatureDisabled(DevFeatures.L2CM);
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
        implementations.l1BlockImpl = address(new L1Block());
        implementations.l1BlockCGTImpl = address(new L1BlockCGT());
        implementations.l2ToL1MessagePasserImpl = address(new L2ToL1MessagePasser());
        implementations.l2ToL1MessagePasserCGTImpl = address(new L2ToL1MessagePasserCGT());
        implementations.optimismMintableERC721FactoryImpl = address(new OptimismMintableERC721Factory());
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
        implementations.conditionalDeployerImpl = deployCode("src/L2/ConditionalDeployer.sol:ConditionalDeployer");
        implementations.l2DevFeatureFlagsImpl = deployCode("src/L2/L2DevFeatureFlags.sol:L2DevFeatureFlags");
    }

    /// @notice Deploys the L2ContractsManager with the loaded implementations.
    function _deployL2CM() internal {
        l2cm = new L2ContractsManager_FunctionsExposer_Harness(implementations);
        vm.label(address(l2cm), "L2ContractsManager");
    }

    /// @notice Executes the upgrade via DELEGATECALL from the L2ProxyAdmin context.
    function _executeUpgrade() internal {
        // The L2CM must be called via DELEGATECALL from the ProxyAdmin.
        // We simulate this by pranking as the ProxyAdmin and using delegatecall.
        address proxyAdmin = Predeploys.PROXY_ADMIN;
        vm.prank(proxyAdmin, true);
        (bool success,) = address(l2cm).delegatecall(abi.encodeCall(L2ContractsManager.upgrade, ()));
        require(success, "L2ContractsManager: Upgrade failed");
    }

    /// @notice Captures the current post-upgrade state of all predeploys.
    /// @return state_ The captured state.
    function _capturePostUpgradeState() internal view returns (PostUpgradeState memory state_) {
        // Capture implementation addresses
        state_.gasPriceOracleImpl = EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE);
        state_.l2CrossDomainMessengerImpl = EIP1967Helper.getImplementation(Predeploys.L2_CROSS_DOMAIN_MESSENGER);
        state_.l2StandardBridgeImpl = EIP1967Helper.getImplementation(Predeploys.L2_STANDARD_BRIDGE);
        state_.sequencerFeeWalletImpl = EIP1967Helper.getImplementation(Predeploys.SEQUENCER_FEE_WALLET);
        state_.optimismMintableERC20FactoryImpl =
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY);
        state_.l2ERC721BridgeImpl = EIP1967Helper.getImplementation(Predeploys.L2_ERC721_BRIDGE);
        state_.l1BlockImpl = EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES);
        state_.l1BlockCGTImpl = EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES);
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
        state_.l2DevFeatureFlagsImpl = EIP1967Helper.getImplementation(Predeploys.L2_DEV_FEATURE_FLAGS);

        // Capture config values using the harness
        state_.config = l2cm.loadFullConfig();
    }

    /// @notice Asserts that two post-upgrade states are identical.
    /// @param _state1 The first state.
    /// @param _state2 The second state.
    function _assertStatesEqual(PostUpgradeState memory _state1, PostUpgradeState memory _state2) internal pure {
        // Assert implementation addresses are equal
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
        assertEq(_state1.l1BlockImpl, _state2.l1BlockImpl, "L1Block impl mismatch");
        assertEq(_state1.l1BlockCGTImpl, _state2.l1BlockCGTImpl, "L1BlockCGT impl mismatch");
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
        assertEq(_state1.l2DevFeatureFlagsImpl, _state2.l2DevFeatureFlagsImpl, "L2DevFeatureFlags impl mismatch");

        // Assert config values are equal
        assertEq(
            address(_state1.config.crossDomainMessenger.otherMessenger),
            address(_state2.config.crossDomainMessenger.otherMessenger),
            "CrossDomainMessenger config mismatch"
        );
        assertEq(
            address(_state1.config.standardBridge.otherBridge),
            address(_state2.config.standardBridge.otherBridge),
            "StandardBridge config mismatch"
        );
        assertEq(
            address(_state1.config.erc721Bridge.otherBridge),
            address(_state2.config.erc721Bridge.otherBridge),
            "ERC721Bridge config mismatch"
        );
        assertEq(
            _state1.config.mintableERC20Factory.bridge,
            _state2.config.mintableERC20Factory.bridge,
            "MintableERC20Factory config mismatch"
        );
        assertEq(
            _state1.config.mintableERC721Factory.bridge,
            _state2.config.mintableERC721Factory.bridge,
            "MintableERC721Factory bridge mismatch"
        );
        assertEq(
            _state1.config.mintableERC721Factory.remoteChainID,
            _state2.config.mintableERC721Factory.remoteChainID,
            "MintableERC721Factory remoteChainID mismatch"
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
            address(_state1.config.feeSplitter.sharesCalculator),
            address(_state2.config.feeSplitter.sharesCalculator),
            "FeeSplitter sharesCalculator mismatch"
        );
    }

    /// @notice Tests that the upgrade produces identical state when called twice with the same pre-state.
    function test_upgradeProducesSameState_whenCalledTwiceWithSamePreState_succeeds() public {
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

    /// @notice Tests that all network-specific configuration is preserved after upgrade.
    function test_upgradePreservesAllConfiguration_succeeds() public {
        // Get the pre-upgrade configuration
        L2ContractsManagerTypes.FullConfig memory preUpgradeConfig = l2cm.loadFullConfig();

        // Execute the upgrade
        _executeUpgrade();

        // Get the post-upgrade configuration from each of the predeploys

        // L2CrossDomainMessenger
        assertEq(
            address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).otherMessenger()),
            address(preUpgradeConfig.crossDomainMessenger.otherMessenger),
            "L2CrossDomainMessenger.otherMessenger not preserved"
        );

        // L2StandardBridge
        assertEq(
            address(IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).otherBridge()),
            address(preUpgradeConfig.standardBridge.otherBridge),
            "L2StandardBridge.otherBridge not preserved"
        );

        // L2ERC721Bridge
        assertEq(
            address(IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).otherBridge()),
            address(preUpgradeConfig.erc721Bridge.otherBridge),
            "L2ERC721Bridge.otherBridge not preserved"
        );

        // OptimismMintableERC20Factory
        assertEq(
            address(IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).bridge()),
            address(preUpgradeConfig.mintableERC20Factory.bridge),
            "OptimismMintableERC20Factory.bridge not preserved"
        );

        // OptimismMintableERC721Factory
        assertEq(
            address(IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).bridge()),
            address(preUpgradeConfig.mintableERC721Factory.bridge),
            "OptimismMintableERC721Factory.bridge not preserved"
        );
        assertEq(
            IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).remoteChainID(),
            preUpgradeConfig.mintableERC721Factory.remoteChainID,
            "OptimismMintableERC721Factory.remoteChainID not preserved"
        );

        // SequencerFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).recipient(),
            address(preUpgradeConfig.sequencerFeeVault.recipient),
            "SequencerFeeVault.recipient not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).minWithdrawalAmount(),
            preUpgradeConfig.sequencerFeeVault.minWithdrawalAmount,
            "SequencerFeeVault.minWithdrawalAmount not preserved"
        );
        assertTrue(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).withdrawalNetwork()
                == preUpgradeConfig.sequencerFeeVault.withdrawalNetwork,
            "SequencerFeeVault.withdrawalNetwork not preserved"
        );

        // BaseFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).recipient(),
            preUpgradeConfig.baseFeeVault.recipient,
            "BaseFeeVault.recipient not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).minWithdrawalAmount(),
            preUpgradeConfig.baseFeeVault.minWithdrawalAmount,
            "BaseFeeVault.minWithdrawalAmount not preserved"
        );
        assertTrue(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).withdrawalNetwork()
                == preUpgradeConfig.baseFeeVault.withdrawalNetwork,
            "BaseFeeVault.withdrawalNetwork not preserved"
        );

        // L1FeeVault
        assertEq(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).recipient(),
            preUpgradeConfig.l1FeeVault.recipient,
            "L1FeeVault.recipient not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).minWithdrawalAmount(),
            preUpgradeConfig.l1FeeVault.minWithdrawalAmount,
            "L1FeeVault.minWithdrawalAmount not preserved"
        );
        assertTrue(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).withdrawalNetwork()
                == preUpgradeConfig.l1FeeVault.withdrawalNetwork,
            "L1FeeVault.withdrawalNetwork not preserved"
        );

        // OperatorFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).recipient(),
            preUpgradeConfig.operatorFeeVault.recipient,
            "OperatorFeeVault.recipient not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).minWithdrawalAmount(),
            preUpgradeConfig.operatorFeeVault.minWithdrawalAmount,
            "OperatorFeeVault.minWithdrawalAmount not preserved"
        );
        assertTrue(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).withdrawalNetwork()
                == preUpgradeConfig.operatorFeeVault.withdrawalNetwork,
            "OperatorFeeVault.withdrawalNetwork not preserved"
        );

        // FeeSplitter
        assertEq(
            address(IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator()),
            address(preUpgradeConfig.feeSplitter.sharesCalculator),
            "FeeSplitter.sharesCalculator not preserved"
        );
    }

    /// @notice Tests that calling upgrade() directly (not via DELEGATECALL) reverts.
    function test_upgrade_whenCalledDirectly_reverts() public {
        // Calling upgrade() directly should revert with OnlyDelegatecall error
        vm.expectRevert(L2ContractsManager.L2ContractsManager_OnlyDelegatecall.selector);
        l2cm.upgrade();
    }

    /// @notice Tests that fee vault configurations with non-default values are preserved after upgrade.
    function test_upgradePreservesFeeVaultConfig_withNonDefaultValues_succeeds() public {
        // Define non-default test values
        address customRecipient = makeAddr("customRecipient");
        uint256 customMinWithdrawal = 50 ether;

        // Get the ProxyAdmin owner
        address proxyAdminOwner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();

        // Set non-default values on all fee vaults before upgrade
        vm.startPrank(proxyAdminOwner);

        // SequencerFeeVault
        IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setRecipient(customRecipient);
        IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setMinWithdrawalAmount(customMinWithdrawal);
        IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);

        // BaseFeeVault
        IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setRecipient(customRecipient);
        IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setMinWithdrawalAmount(customMinWithdrawal);
        IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);

        // L1FeeVault
        IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setRecipient(customRecipient);
        IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setMinWithdrawalAmount(customMinWithdrawal);
        IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);

        // OperatorFeeVault
        IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setRecipient(customRecipient);
        IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setMinWithdrawalAmount(customMinWithdrawal);
        IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);

        vm.stopPrank();

        // Execute the upgrade
        _executeUpgrade();

        // Verify non-default values are preserved on all fee vaults

        // SequencerFeeVault
        _assertFeeVaultConfig(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)),
            customRecipient,
            customMinWithdrawal,
            Types.WithdrawalNetwork.L2
        );

        // BaseFeeVault
        _assertFeeVaultConfig(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)),
            customRecipient,
            customMinWithdrawal,
            Types.WithdrawalNetwork.L2
        );
        // L1FeeVault
        _assertFeeVaultConfig(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)),
            customRecipient,
            customMinWithdrawal,
            Types.WithdrawalNetwork.L2
        );
        // OperatorFeeVault
        _assertFeeVaultConfig(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)),
            customRecipient,
            customMinWithdrawal,
            Types.WithdrawalNetwork.L2
        );
    }

    function _assertFeeVaultConfig(
        IFeeVault _feeVault,
        address _expectedRecipient,
        uint256 _expectedMinWithdrawalAmount,
        Types.WithdrawalNetwork _expectedWithdrawalNetwork
    )
        internal
        view
    {
        assertEq(_feeVault.recipient(), _expectedRecipient, "FeeVault.recipient not preserved");
        assertEq(
            _feeVault.minWithdrawalAmount(), _expectedMinWithdrawalAmount, "FeeVault.minWithdrawalAmount not preserved"
        );
        assertTrue(
            _feeVault.withdrawalNetwork() == _expectedWithdrawalNetwork, "FeeVault.withdrawalNetwork not preserved"
        );
    }
}

/// @title L2ContractsManager_CGT_Test
/// @notice Test contract for the L2ContractsManager on Custom Gas Token networks.
contract L2ContractsManager_Upgrade_CGT_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Tests that CGT-specific contracts are upgraded when CGT is enabled.
    function test_upgradeUpgradesCGTContracts_whenCGTEnabled_succeeds() public {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        // Capture pre-upgrade implementations for CGT-specific contracts
        address preUpgradeLiquidityControllerImpl = EIP1967Helper.getImplementation(Predeploys.LIQUIDITY_CONTROLLER);
        address preUpgradeNativeAssetLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.NATIVE_ASSET_LIQUIDITY);

        // Execute the upgrade
        _executeUpgrade();

        // Verify LiquidityController was upgraded
        address postUpgradeLiquidityControllerImpl = EIP1967Helper.getImplementation(Predeploys.LIQUIDITY_CONTROLLER);
        assertEq(
            postUpgradeLiquidityControllerImpl,
            implementations.liquidityControllerImpl,
            "LiquidityController should be upgraded to new implementation"
        );
        assertTrue(
            postUpgradeLiquidityControllerImpl != preUpgradeLiquidityControllerImpl
                || preUpgradeLiquidityControllerImpl == implementations.liquidityControllerImpl,
            "LiquidityController implementation should change or already be target"
        );

        // Verify NativeAssetLiquidity was upgraded
        address postUpgradeNativeAssetLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.NATIVE_ASSET_LIQUIDITY);
        assertEq(
            postUpgradeNativeAssetLiquidityImpl,
            implementations.nativeAssetLiquidityImpl,
            "NativeAssetLiquidity should be upgraded to new implementation"
        );
        assertTrue(
            postUpgradeNativeAssetLiquidityImpl != preUpgradeNativeAssetLiquidityImpl
                || preUpgradeNativeAssetLiquidityImpl == implementations.nativeAssetLiquidityImpl,
            "NativeAssetLiquidity implementation should change or already be target"
        );

        // Verify L1Block uses CGT implementation
        address postUpgradeL1BlockImpl = EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES);
        assertEq(
            postUpgradeL1BlockImpl,
            implementations.l1BlockCGTImpl,
            "L1Block should use CGT implementation on CGT networks"
        );

        // Verify L2ToL1MessagePasser uses CGT implementation
        address postUpgradeL2ToL1MessagePasserImpl = EIP1967Helper.getImplementation(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        assertEq(
            postUpgradeL2ToL1MessagePasserImpl,
            implementations.l2ToL1MessagePasserCGTImpl,
            "L2ToL1MessagePasser should use CGT implementation on CGT networks"
        );
    }

    /// @notice Tests that LiquidityController config is preserved after upgrade on CGT networks.
    function test_upgradePreservesLiquidityControllerConfig_onCGTNetwork_succeeds() public {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        // Capture pre-upgrade config
        L2ContractsManagerTypes.FullConfig memory preUpgradeConfig = l2cm.loadFullConfig();

        // Execute the upgrade
        _executeUpgrade();

        // Verify LiquidityController config is preserved
        ILiquidityController liquidityController = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
        assertEq(
            liquidityController.owner(),
            preUpgradeConfig.liquidityController.owner,
            "LiquidityController.owner not preserved"
        );
        assertEq(
            liquidityController.gasPayingTokenName(),
            preUpgradeConfig.liquidityController.gasPayingTokenName,
            "LiquidityController.gasPayingTokenName not preserved"
        );
        assertEq(
            liquidityController.gasPayingTokenSymbol(),
            preUpgradeConfig.liquidityController.gasPayingTokenSymbol,
            "LiquidityController.gasPayingTokenSymbol not preserved"
        );
    }
}

/// @title L2ContractsManager_Upgrade_DowngradePrevention_Test
/// @notice Test contract that verifies L2CM prevents downgrading predeploy implementations.
contract L2ContractsManager_Upgrade_DowngradePrevention_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Tests that upgrade reverts when a non-initializable predeploy has a higher version than the new
    /// implementation.
    function test_upgrade_whenDowngradingNonInitializablePredeploy_reverts() public {
        // Mock GasPriceOracle to report a version higher than the new implementation
        string memory higherVersion = "999.0.0";
        vm.mockCall(Predeploys.GAS_PRICE_ORACLE, abi.encodeCall(ISemver.version, ()), abi.encode(higherVersion));

        vm.expectRevert(
            abi.encodeWithSelector(
                L2ContractsManagerUtils.L2ContractsManager_DowngradeNotAllowed.selector, Predeploys.GAS_PRICE_ORACLE
            )
        );
        _executeUpgrade();
    }

    /// @notice Tests that upgrade reverts when an initializable predeploy has a higher version than the new
    /// implementation.
    function test_upgrade_whenDowngradingInitializablePredeploy_reverts() public {
        // Mock L2CrossDomainMessenger to report a version higher than the new implementation
        string memory higherVersion = "999.0.0";
        vm.mockCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER, abi.encodeCall(ISemver.version, ()), abi.encode(higherVersion)
        );

        vm.expectRevert(
            abi.encodeWithSelector(
                L2ContractsManagerUtils.L2ContractsManager_DowngradeNotAllowed.selector,
                Predeploys.L2_CROSS_DOMAIN_MESSENGER
            )
        );
        _executeUpgrade();
    }

    /// @notice Tests that upgrade succeeds when the predeploy has the same version as the new implementation
    /// (not a downgrade).
    function test_upgrade_whenSameVersion_succeeds() public {
        // Mock GasPriceOracle to report the same version as the new implementation
        string memory implVersion = ISemver(implementations.gasPriceOracleImpl).version();
        vm.mockCall(Predeploys.GAS_PRICE_ORACLE, abi.encodeCall(ISemver.version, ()), abi.encode(implVersion));

        _executeUpgrade();

        // Verify the upgrade went through
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE),
            implementations.gasPriceOracleImpl,
            "GasPriceOracle should be upgraded"
        );
    }
}

/// @title L2ContractsManager_GetImplementations_Test
/// @notice Tests for the getImplementations() getter function.
contract L2ContractsManager_GetImplementations_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Tests that getImplementations returns all implementation addresses matching the constructor input.
    function test_getImplementations_returnsAllImplementations_succeeds() public view {
        L2ContractsManagerTypes.Implementations memory result = l2cm.getImplementations();

        assertEq(result.storageSetterImpl, implementations.storageSetterImpl, "storageSetterImpl mismatch");
        assertEq(
            result.l2CrossDomainMessengerImpl,
            implementations.l2CrossDomainMessengerImpl,
            "l2CrossDomainMessengerImpl mismatch"
        );
        assertEq(result.gasPriceOracleImpl, implementations.gasPriceOracleImpl, "gasPriceOracleImpl mismatch");
        assertEq(result.l2StandardBridgeImpl, implementations.l2StandardBridgeImpl, "l2StandardBridgeImpl mismatch");
        assertEq(
            result.sequencerFeeWalletImpl, implementations.sequencerFeeWalletImpl, "sequencerFeeWalletImpl mismatch"
        );
        assertEq(
            result.optimismMintableERC20FactoryImpl,
            implementations.optimismMintableERC20FactoryImpl,
            "optimismMintableERC20FactoryImpl mismatch"
        );
        assertEq(result.l2ERC721BridgeImpl, implementations.l2ERC721BridgeImpl, "l2ERC721BridgeImpl mismatch");
        assertEq(result.l1BlockImpl, implementations.l1BlockImpl, "l1BlockImpl mismatch");
        assertEq(result.l1BlockCGTImpl, implementations.l1BlockCGTImpl, "l1BlockCGTImpl mismatch");
        assertEq(
            result.l2ToL1MessagePasserImpl, implementations.l2ToL1MessagePasserImpl, "l2ToL1MessagePasserImpl mismatch"
        );
        assertEq(
            result.l2ToL1MessagePasserCGTImpl,
            implementations.l2ToL1MessagePasserCGTImpl,
            "l2ToL1MessagePasserCGTImpl mismatch"
        );
        assertEq(
            result.optimismMintableERC721FactoryImpl,
            implementations.optimismMintableERC721FactoryImpl,
            "optimismMintableERC721FactoryImpl mismatch"
        );
        assertEq(result.proxyAdminImpl, implementations.proxyAdminImpl, "proxyAdminImpl mismatch");
        assertEq(result.baseFeeVaultImpl, implementations.baseFeeVaultImpl, "baseFeeVaultImpl mismatch");
        assertEq(result.l1FeeVaultImpl, implementations.l1FeeVaultImpl, "l1FeeVaultImpl mismatch");
        assertEq(result.operatorFeeVaultImpl, implementations.operatorFeeVaultImpl, "operatorFeeVaultImpl mismatch");
        assertEq(result.schemaRegistryImpl, implementations.schemaRegistryImpl, "schemaRegistryImpl mismatch");
        assertEq(result.easImpl, implementations.easImpl, "easImpl mismatch");
        assertEq(result.crossL2InboxImpl, implementations.crossL2InboxImpl, "crossL2InboxImpl mismatch");
        assertEq(
            result.l2ToL2CrossDomainMessengerImpl,
            implementations.l2ToL2CrossDomainMessengerImpl,
            "l2ToL2CrossDomainMessengerImpl mismatch"
        );
        assertEq(
            result.superchainETHBridgeImpl, implementations.superchainETHBridgeImpl, "superchainETHBridgeImpl mismatch"
        );
        assertEq(result.ethLiquidityImpl, implementations.ethLiquidityImpl, "ethLiquidityImpl mismatch");
        assertEq(
            result.optimismSuperchainERC20FactoryImpl,
            implementations.optimismSuperchainERC20FactoryImpl,
            "optimismSuperchainERC20FactoryImpl mismatch"
        );
        assertEq(
            result.optimismSuperchainERC20BeaconImpl,
            implementations.optimismSuperchainERC20BeaconImpl,
            "optimismSuperchainERC20BeaconImpl mismatch"
        );
        assertEq(
            result.superchainTokenBridgeImpl,
            implementations.superchainTokenBridgeImpl,
            "superchainTokenBridgeImpl mismatch"
        );
        assertEq(
            result.nativeAssetLiquidityImpl,
            implementations.nativeAssetLiquidityImpl,
            "nativeAssetLiquidityImpl mismatch"
        );
        assertEq(
            result.liquidityControllerImpl, implementations.liquidityControllerImpl, "liquidityControllerImpl mismatch"
        );
        assertEq(result.feeSplitterImpl, implementations.feeSplitterImpl, "feeSplitterImpl mismatch");
        assertEq(
            result.conditionalDeployerImpl, implementations.conditionalDeployerImpl, "conditionalDeployerImpl mismatch"
        );
        assertEq(result.l2DevFeatureFlagsImpl, implementations.l2DevFeatureFlagsImpl, "l2DevFeatureFlagsImpl mismatch");
    }

    /// @notice Tests that no field in getImplementations() is left uninitialized
    ///         when all implementations are provided to the constructor.
    function test_getImplementations_noFieldIsZero_succeeds() public view {
        L2ContractsManagerTypes.Implementations memory result = l2cm.getImplementations();

        assertTrue(result.storageSetterImpl != address(0), "storageSetterImpl is zero");
        assertTrue(result.l2CrossDomainMessengerImpl != address(0), "l2CrossDomainMessengerImpl is zero");
        assertTrue(result.gasPriceOracleImpl != address(0), "gasPriceOracleImpl is zero");
        assertTrue(result.l2StandardBridgeImpl != address(0), "l2StandardBridgeImpl is zero");
        assertTrue(result.sequencerFeeWalletImpl != address(0), "sequencerFeeWalletImpl is zero");
        assertTrue(result.optimismMintableERC20FactoryImpl != address(0), "optimismMintableERC20FactoryImpl is zero");
        assertTrue(result.l2ERC721BridgeImpl != address(0), "l2ERC721BridgeImpl is zero");
        assertTrue(result.l1BlockImpl != address(0), "l1BlockImpl is zero");
        assertTrue(result.l1BlockCGTImpl != address(0), "l1BlockCGTImpl is zero");
        assertTrue(result.l2ToL1MessagePasserImpl != address(0), "l2ToL1MessagePasserImpl is zero");
        assertTrue(result.l2ToL1MessagePasserCGTImpl != address(0), "l2ToL1MessagePasserCGTImpl is zero");
        assertTrue(result.optimismMintableERC721FactoryImpl != address(0), "optimismMintableERC721FactoryImpl is zero");
        assertTrue(result.proxyAdminImpl != address(0), "proxyAdminImpl is zero");
        assertTrue(result.baseFeeVaultImpl != address(0), "baseFeeVaultImpl is zero");
        assertTrue(result.l1FeeVaultImpl != address(0), "l1FeeVaultImpl is zero");
        assertTrue(result.operatorFeeVaultImpl != address(0), "operatorFeeVaultImpl is zero");
        assertTrue(result.schemaRegistryImpl != address(0), "schemaRegistryImpl is zero");
        assertTrue(result.easImpl != address(0), "easImpl is zero");
        assertTrue(result.crossL2InboxImpl != address(0), "crossL2InboxImpl is zero");
        assertTrue(result.l2ToL2CrossDomainMessengerImpl != address(0), "l2ToL2CrossDomainMessengerImpl is zero");
        assertTrue(result.superchainETHBridgeImpl != address(0), "superchainETHBridgeImpl is zero");
        assertTrue(result.ethLiquidityImpl != address(0), "ethLiquidityImpl is zero");
        assertTrue(
            result.optimismSuperchainERC20FactoryImpl != address(0), "optimismSuperchainERC20FactoryImpl is zero"
        );
        assertTrue(result.optimismSuperchainERC20BeaconImpl != address(0), "optimismSuperchainERC20BeaconImpl is zero");
        assertTrue(result.superchainTokenBridgeImpl != address(0), "superchainTokenBridgeImpl is zero");
        assertTrue(result.nativeAssetLiquidityImpl != address(0), "nativeAssetLiquidityImpl is zero");
        assertTrue(result.liquidityControllerImpl != address(0), "liquidityControllerImpl is zero");
        assertTrue(result.feeSplitterImpl != address(0), "feeSplitterImpl is zero");
        assertTrue(result.conditionalDeployerImpl != address(0), "conditionalDeployerImpl is zero");
        assertTrue(result.l2DevFeatureFlagsImpl != address(0), "l2DevFeatureFlagsImpl is zero");
    }
}

/// @title L2ContractsManager_Upgrade_InteropFlag_Test
/// @notice Tests that interop predeploy upgrades are correctly gated behind the OPTIMISM_PORTAL_INTEROP dev feature
/// flag.
contract L2ContractsManager_Upgrade_InteropFlag_Test is L2ContractsManager_Upgrade_Test {
    /// @notice The list of interop predeploy addresses.
    address[] internal interopPredeploys;

    function setUp() public override {
        super.setUp();
        interopPredeploys.push(Predeploys.CROSS_L2_INBOX);
        interopPredeploys.push(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        interopPredeploys.push(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        interopPredeploys.push(Predeploys.ETH_LIQUIDITY);
        interopPredeploys.push(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY);
        interopPredeploys.push(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON);
        interopPredeploys.push(Predeploys.SUPERCHAIN_TOKEN_BRIDGE);
    }

    /// @notice Tests that all 7 interop predeploys are upgraded when OPTIMISM_PORTAL_INTEROP flag is enabled.
    function test_upgradeUpgradesInteropPredeploys_whenInteropFlagEnabled_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Capture pre-upgrade implementations
        address[] memory preUpgradeImpls = new address[](interopPredeploys.length);
        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            preUpgradeImpls[i] = EIP1967Helper.getImplementation(interopPredeploys[i]);
        }

        _executeUpgrade();

        // Verify all interop predeploys were upgraded to new implementations
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.CROSS_L2_INBOX),
            implementations.crossL2InboxImpl,
            "CrossL2Inbox should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER),
            implementations.l2ToL2CrossDomainMessengerImpl,
            "L2ToL2CrossDomainMessenger should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.SUPERCHAIN_ETH_BRIDGE),
            implementations.superchainETHBridgeImpl,
            "SuperchainETHBridge should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.ETH_LIQUIDITY),
            implementations.ethLiquidityImpl,
            "ETHLiquidity should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY),
            implementations.optimismSuperchainERC20FactoryImpl,
            "OptimismSuperchainERC20Factory should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON),
            implementations.optimismSuperchainERC20BeaconImpl,
            "OptimismSuperchainERC20Beacon should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.SUPERCHAIN_TOKEN_BRIDGE),
            implementations.superchainTokenBridgeImpl,
            "SuperchainTokenBridge should be upgraded"
        );
    }

    /// @notice Tests that all 7 interop predeploys retain pre-upgrade implementations when OPTIMISM_PORTAL_INTEROP flag
    /// is disabled.
    function test_upgradeSkipsInteropPredeploys_whenInteropFlagDisabled_succeeds() public {
        skipIfDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Capture pre-upgrade implementations
        address[] memory preUpgradeImpls = new address[](interopPredeploys.length);
        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            preUpgradeImpls[i] = EIP1967Helper.getImplementation(interopPredeploys[i]);
        }

        _executeUpgrade();

        // Verify all interop predeploys were NOT upgraded (still have pre-upgrade implementations)
        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            assertEq(
                EIP1967Helper.getImplementation(interopPredeploys[i]),
                preUpgradeImpls[i],
                "Interop predeploy should not be upgraded when OPTIMISM_PORTAL_INTEROP is disabled"
            );
        }
    }
}

/// @title L2ContractsManager_Upgrade_Coverage_Test
/// @notice Test that verifies all predeploys receive upgrade calls during L2CM upgrade.
///         Uses Predeploys.sol as the source of truth for which predeploys should be upgraded.
contract L2ContractsManager_Upgrade_Coverage_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Checks if a predeploy is an interop predeploy gated behind the OPTIMISM_PORTAL_INTEROP dev feature flag.
    function _isInteropPredeploy(address _predeploy) internal pure returns (bool) {
        return _predeploy == Predeploys.CROSS_L2_INBOX || _predeploy == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER
            || _predeploy == Predeploys.SUPERCHAIN_ETH_BRIDGE || _predeploy == Predeploys.ETH_LIQUIDITY
            || _predeploy == Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY
            || _predeploy == Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON || _predeploy == Predeploys.SUPERCHAIN_TOKEN_BRIDGE;
    }

    /// @notice Returns CGT-only predeploys that require initialization.
    /// @dev These are separate because they're only deployed on CGT networks.
    function _getCGTInitializablePredeploys() internal pure returns (address[] memory predeploys_) {
        predeploys_ = new address[](1);
        predeploys_[0] = Predeploys.LIQUIDITY_CONTROLLER;
    }

    /// @notice Checks if a predeploy requires initialization.
    /// @dev Returns true for predeploys that have an initializer and need upgradeToAndCall.
    ///      This determines the upgrade method, not coverage.
    function _requiresInitialization(address _predeploy) internal pure returns (bool) {
        return _predeploy == Predeploys.L2_CROSS_DOMAIN_MESSENGER || _predeploy == Predeploys.L2_STANDARD_BRIDGE
            || _predeploy == Predeploys.L2_ERC721_BRIDGE || _predeploy == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY
            || _predeploy == Predeploys.SEQUENCER_FEE_WALLET || _predeploy == Predeploys.BASE_FEE_VAULT
            || _predeploy == Predeploys.L1_FEE_VAULT || _predeploy == Predeploys.OPERATOR_FEE_VAULT
            || _predeploy == Predeploys.FEE_SPLITTER || _predeploy == Predeploys.LIQUIDITY_CONTROLLER;
    }

    /// @notice Checks if a predeploy is deployed and upgradeable.
    /// @dev Uses EIP1967Helper to read the implementation slot directly from storage.
    ///      This avoids calling the proxy's implementation() function which may fail.
    function _isPredeployUpgradeable(address _proxy) internal view returns (bool) {
        address impl = EIP1967Helper.getImplementation(_proxy);
        return impl != address(0) && impl.code.length > 0;
    }

    /// @notice Tests that all predeploys from Predeploys.sol receive the expected upgrade call.
    ///         Uses vm.expectCall() to verify that upgradeTo or upgradeToAndCall is called.
    /// @dev If L2CM misses a predeploy that exists in Predeploys.sol, this test will fail.
    function test_allPredeploysReceiveUpgradeCall_succeeds() public {
        address[] memory allPredeploys = Predeploys.getUpgradeablePredeploys();
        bool interopEnabled = isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        for (uint256 i = 0; i < allPredeploys.length; i++) {
            address predeploy = allPredeploys[i];

            // Skip predeploys that are not deployed on this chain (e.g., CGT-only, interop-only)
            if (!_isPredeployUpgradeable(predeploy)) continue;

            // Skip interop predeploys when OPTIMISM_PORTAL_INTEROP flag is disabled
            if (_isInteropPredeploy(predeploy) && !interopEnabled) continue;

            // Expect the appropriate upgrade call based on whether initialization is required
            if (_requiresInitialization(predeploy)) {
                // nosemgrep:sol-style-use-abi-encodecall
                vm.expectCall(predeploy, abi.encodeWithSelector(IProxy.upgradeToAndCall.selector));
            } else {
                // nosemgrep:sol-style-use-abi-encodecall
                vm.expectCall(predeploy, abi.encodeWithSelector(IProxy.upgradeTo.selector));
            }
        }

        _executeUpgrade();
    }

    /// @notice Tests that CGT-specific predeploys receive upgrade calls on CGT networks.
    /// @dev CGT predeploys are conditionally deployed, so they need separate verification.
    function test_cgtPredeploysReceiveUpgradeCall_whenCGTEnabled_succeeds() public {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        // Get CGT-only predeploys that require initialization
        address[] memory cgtInitPredeploys = _getCGTInitializablePredeploys();
        for (uint256 i = 0; i < cgtInitPredeploys.length; i++) {
            // nosemgrep:sol-style-use-abi-encodecall
            vm.expectCall(cgtInitPredeploys[i], abi.encodeWithSelector(IProxy.upgradeToAndCall.selector));
        }

        // NativeAssetLiquidity uses upgradeTo
        // nosemgrep:sol-style-use-abi-encodecall
        vm.expectCall(Predeploys.NATIVE_ASSET_LIQUIDITY, abi.encodeWithSelector(IProxy.upgradeTo.selector));

        _executeUpgrade();
    }
}

/// @title L2ContractsManager_Upgrade_NullSafeFlagsImpl_Test
/// @notice Tests the upgrade process when the L2DevFeatureFlags
///         implementation has no code, simulating existing chains where the predeploy was
///         never deployed.
contract L2ContractsManager_Upgrade_NullSafeFlagsImpl_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Helper function that simulates an existing-chain state where L2DevFeatureFlags has not been deployed by:
    ///         1. Etching the current implementation to have no code.
    ///         2. Pointing implementations.l2DevFeatureFlagsImpl to a fresh address with no code,
    ///            so that after the upgrade sets the new impl pointer, the null-safe guard in
    ///            _isDevFeatureEnabled fires and returns false.
    function _simulateNoFlagsImpl() internal {
        address currentImpl = EIP1967Helper.getImplementation(Predeploys.L2_DEV_FEATURE_FLAGS);
        vm.etch(currentImpl, bytes(""));

        implementations.l2DevFeatureFlagsImpl = makeAddr("emptyFlagsImpl");
        _deployL2CM();
    }

    /// @notice Tests that _isDevFeatureEnabled returns false when the flags implementation has no code.
    function testFuzz_isDevFeatureEnabled_whenFlagsImplHasNoCode_succeeds(bytes32 _feature) public {
        _simulateNoFlagsImpl();
        assertFalse(l2cm.isDevFeatureEnabled(_feature));
    }

    /// @notice Tests that the upgrade does not revert when L2DevFeatureFlags implementation has no code.
    function test_upgrade_whenFlagsImplHasNoCode_succeeds() public {
        _simulateNoFlagsImpl();
        _executeUpgrade();
    }

    /// @notice Tests that all 7 interop predeploys retain their pre-upgrade implementations
    ///         when the flags implementation has no code.
    function test_upgrade_skipsInteropPredeploys_succeeds() public {
        address[] memory interopPredeploys = new address[](7);
        interopPredeploys[0] = Predeploys.CROSS_L2_INBOX;
        interopPredeploys[1] = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
        interopPredeploys[2] = Predeploys.SUPERCHAIN_ETH_BRIDGE;
        interopPredeploys[3] = Predeploys.ETH_LIQUIDITY;
        interopPredeploys[4] = Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY;
        interopPredeploys[5] = Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON;
        interopPredeploys[6] = Predeploys.SUPERCHAIN_TOKEN_BRIDGE;

        address[] memory preUpgradeImpls = new address[](7);
        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            preUpgradeImpls[i] = EIP1967Helper.getImplementation(interopPredeploys[i]);
        }

        _simulateNoFlagsImpl();
        _executeUpgrade();

        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            assertEq(
                EIP1967Helper.getImplementation(interopPredeploys[i]),
                preUpgradeImpls[i],
                "Interop predeploy should not be upgraded when flags impl has no code"
            );
        }
    }

    /// @notice Tests that non-interop predeploys are still upgraded when L2DevFeatureFlags
    ///         implementation has no code.
    function test_upgrade_upgradesNonInteropPredeploys_succeeds() public {
        _simulateNoFlagsImpl();
        _executeUpgrade();

        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L2_CROSS_DOMAIN_MESSENGER),
            implementations.l2CrossDomainMessengerImpl,
            "L2CrossDomainMessenger should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE),
            implementations.gasPriceOracleImpl,
            "GasPriceOracle should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES),
            Config.sysFeatureCustomGasToken() ? implementations.l1BlockCGTImpl : implementations.l1BlockImpl,
            "L1Block should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L2_STANDARD_BRIDGE),
            implementations.l2StandardBridgeImpl,
            "L2StandardBridge should be upgraded"
        );
    }
}
