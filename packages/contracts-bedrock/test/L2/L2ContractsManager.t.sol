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
import { Types } from "src/libraries/Types.sol";
import { Features } from "src/libraries/Features.sol";
import { Config } from "scripts/libraries/Config.sol";
import { LibString } from "@solady/utils/LibString.sol";
import { stdStorage, StdStorage } from "forge-std/StdStorage.sol";
import { console } from "forge-std/console.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IProxy } from "interfaces/universal/IProxy.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";

/// @title L2ContractsManager_FunctionsExposer_Harness
/// @notice Harness contract that exposes internal functions for testing.
contract L2ContractsManager_FunctionsExposer_Harness is L2ContractsManager {
    constructor(L2ContractsManagerTypes.ImplRecord[] memory _implementations) L2ContractsManager(_implementations) { }

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
    error ImplNotFound(string name);

    L2ContractsManager_FunctionsExposer_Harness internal l2cm;
    L2ContractsManagerTypes.ImplRecord[] internal _implRecords;

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
        address nativeAssetLiquidityImpl;
        address liquidityControllerImpl;
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

    /// @notice Looks up an implementation address by name in _implRecords.
    function _findImplByName(string memory _name) internal view returns (address) {
        for (uint256 i = 0; i < _implRecords.length; i++) {
            if (LibString.eq(_implRecords[i].name, _name)) return _implRecords[i].impl;
        }
        revert ImplNotFound(_name);
    }

    /// @notice Deploys the target implementations for the predeploys using the registry as the source of truth.
    function _loadImplementations() internal {
        // StorageSetter is not a predeploy — add it separately.
        _implRecords.push(
            L2ContractsManagerTypes.ImplRecord({ name: "StorageSetter", impl: address(new StorageSetter()) })
        );

        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();
        for (uint256 i = 0; i < records.length; i++) {
            _implRecords.push(
                L2ContractsManagerTypes.ImplRecord({ name: records[i].name, impl: deployCode(records[i].artifactPath) })
            );
        }
    }

    /// @notice Deploys the L2ContractsManager with the loaded implementations.
    function _deployL2CM() internal {
        l2cm = new L2ContractsManager_FunctionsExposer_Harness(_implRecords);
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
        state_.nativeAssetLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.NATIVE_ASSET_LIQUIDITY);
        state_.liquidityControllerImpl = EIP1967Helper.getImplementation(Predeploys.LIQUIDITY_CONTROLLER);
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
            _state1.nativeAssetLiquidityImpl, _state2.nativeAssetLiquidityImpl, "NativeAssetLiquidity impl mismatch"
        );
        assertEq(_state1.liquidityControllerImpl, _state2.liquidityControllerImpl, "LiquidityController impl mismatch");
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
        address proxyAdminOwner = IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner();

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

    /// @notice Checks if a predeploy is deployed and upgradeable.
    /// @dev Uses EIP1967Helper to read the implementation slot directly from storage.
    ///      This avoids calling the proxy's implementation() function which may fail.
    function _isPredeployUpgradeable(address _proxy) internal view returns (bool) {
        address impl = EIP1967Helper.getImplementation(_proxy);
        return impl != address(0) && impl.code.length > 0;
    }

    /// @notice Checks if a predeploy requires initialization.
    /// @dev Returns true for predeploys that have an initializer and need upgradeToAndCall.
    ///      This determines the upgrade method, not coverage.
    function _requiresInitialization(address _predeploy) internal pure returns (bool) {
        return _predeploy == Predeploys.L2_CROSS_DOMAIN_MESSENGER || _predeploy == Predeploys.L2_STANDARD_BRIDGE
            || _predeploy == Predeploys.L2_ERC721_BRIDGE || _predeploy == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY
            || _predeploy == Predeploys.SEQUENCER_FEE_WALLET || _predeploy == Predeploys.BASE_FEE_VAULT
            || _predeploy == Predeploys.L1_FEE_VAULT || _predeploy == Predeploys.OPERATOR_FEE_VAULT
            || _predeploy == Predeploys.LIQUIDITY_CONTROLLER || _predeploy == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY;
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
            _findImplByName("LiquidityController"),
            "LiquidityController should be upgraded to new implementation"
        );
        assertTrue(
            postUpgradeLiquidityControllerImpl != preUpgradeLiquidityControllerImpl
                || preUpgradeLiquidityControllerImpl == _findImplByName("LiquidityController"),
            "LiquidityController implementation should change or already be target"
        );

        // Verify NativeAssetLiquidity was upgraded
        address postUpgradeNativeAssetLiquidityImpl = EIP1967Helper.getImplementation(Predeploys.NATIVE_ASSET_LIQUIDITY);
        assertEq(
            postUpgradeNativeAssetLiquidityImpl,
            _findImplByName("NativeAssetLiquidity"),
            "NativeAssetLiquidity should be upgraded to new implementation"
        );
        assertTrue(
            postUpgradeNativeAssetLiquidityImpl != preUpgradeNativeAssetLiquidityImpl
                || preUpgradeNativeAssetLiquidityImpl == _findImplByName("NativeAssetLiquidity"),
            "NativeAssetLiquidity implementation should change or already be target"
        );

        // Verify L1Block uses CGT implementation
        address postUpgradeL1BlockImpl = EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES);
        assertEq(
            postUpgradeL1BlockImpl,
            _findImplByName("L1BlockCGT"),
            "L1Block should use CGT implementation on CGT networks"
        );

        // Verify L2ToL1MessagePasser uses CGT implementation
        address postUpgradeL2ToL1MessagePasserImpl = EIP1967Helper.getImplementation(Predeploys.L2_TO_L1_MESSAGE_PASSER);
        assertEq(
            postUpgradeL2ToL1MessagePasserImpl,
            _findImplByName("L2ToL1MessagePasserCGT"),
            "L2ToL1MessagePasser should use CGT implementation on CGT networks"
        );
    }

    /// @notice Tests that upgrade succeeds when LiquidityController.owner() is address(0) on CGT networks.
    ///         L2CM replays the live owner back into LiquidityController.initialize(), so a renounced
    ///         ownership state must be preserved across upgrades rather than blocking them.
    function test_upgrade_whenLiquidityControllerOwnerIsZero_succeeds() public {
        skipIfSysFeatureDisabled(Features.CUSTOM_GAS_TOKEN);

        vm.mockCall(
            Predeploys.LIQUIDITY_CONTROLLER, abi.encodeCall(ILiquidityController.owner, ()), abi.encode(address(0))
        );

        // Upgrade must not revert even though owner is address(0)
        _executeUpgrade();

        assertEq(ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER).owner(), address(0));
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
        string memory implVersion = ISemver(_findImplByName("GasPriceOracle")).version();
        vm.mockCall(Predeploys.GAS_PRICE_ORACLE, abi.encodeCall(ISemver.version, ()), abi.encode(implVersion));

        _executeUpgrade();

        // Verify the upgrade went through
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE),
            _findImplByName("GasPriceOracle"),
            "GasPriceOracle should be upgraded"
        );
    }
}

/// @title L2ContractsManager_GetImplementations_Test
/// @notice Tests for the getImplementations() getter function.
contract L2ContractsManager_GetImplementations_Test is L2ContractsManager_Upgrade_Test {
    /// @notice Tests that the ImplRecord[] passed to the constructor is well-formed:
    ///         every entry has a non-empty name and a non-zero implementation address.
    function test_implRecords_areWellFormed_succeeds() public view {
        L2ContractsManagerTypes.ImplRecord[] memory implementationRecords = l2cm.getImplementations();
        // 1 StorageSetter + one entry per upgradeable predeploy record (including variants).
        assertEq(
            implementationRecords.length,
            Predeploys.getUpgradeableRecords().length + 1,
            "ImplRecord count must equal upgradeable predeploy count + 1 (StorageSetter)"
        );
        for (uint256 i = 0; i < implementationRecords.length; i++) {
            assertTrue(bytes(implementationRecords[i].name).length > 0, "ImplRecord name is empty");
            assertTrue(
                implementationRecords[i].impl != address(0),
                string.concat(implementationRecords[i].name, " impl is zero")
            );
        }
    }

    /// @notice Tests that getImplementations returns all implementation addresses matching the constructor input.
    function test_getImplementations_returnsAllImplementations_succeeds() public view {
        L2ContractsManagerTypes.ImplRecord[] memory result = l2cm.getImplementations();

        assertEq(result.length, _implRecords.length, "getImplementations length mismatch");
        for (uint256 i = 0; i < result.length; i++) {
            assertTrue(result[i].impl != address(0), string.concat(result[i].name, " impl is zero"));
            assertEq(result[i].impl, _findImplByName(result[i].name), string.concat(result[i].name, " impl mismatch"));
        }
    }
}

/// @title L2ContractsManager_Upgrade_InteropFlagEnabled_Test
/// @notice Tests that interop predeploy upgrades are correctly gated behind the INTEROP sys feature (set on L1Block).
contract L2ContractsManager_Upgrade_InteropFlagEnabled_Test is L2ContractsManager_Upgrade_Test {
    using stdStorage for StdStorage;

    /// @notice The list of interop predeploy addresses.
    address[] internal interopPredeploys;

    function setUp() public override {
        super.enableInterop();
        super.setUp();
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);
        interopPredeploys.push(Predeploys.CROSS_L2_INBOX);
        interopPredeploys.push(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        interopPredeploys.push(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        interopPredeploys.push(Predeploys.ETH_LIQUIDITY);
    }

    /// @notice Tests that all 4 interop predeploys are upgraded when the INTEROP sys feature is enabled
    ///         (which requires OPTIMISM_PORTAL_INTEROP dev feature to also be enabled for consistency).
    function test_upgradeUpgradesInteropPredeploys_whenInteropFlagEnabled_succeeds() public {
        // Capture pre-upgrade implementations
        address[] memory preUpgradeImpls = new address[](interopPredeploys.length);
        for (uint256 i = 0; i < interopPredeploys.length; i++) {
            preUpgradeImpls[i] = EIP1967Helper.getImplementation(interopPredeploys[i]);
        }

        _executeUpgrade();

        // Verify all interop predeploys were upgraded to new implementations
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.CROSS_L2_INBOX),
            _findImplByName("CrossL2Inbox"),
            "CrossL2Inbox should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER),
            _findImplByName("L2ToL2CrossDomainMessenger"),
            "L2ToL2CrossDomainMessenger should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.SUPERCHAIN_ETH_BRIDGE),
            _findImplByName("SuperchainETHBridge"),
            "SuperchainETHBridge should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.ETH_LIQUIDITY),
            _findImplByName("ETHLiquidity"),
            "ETHLiquidity should be upgraded"
        );
    }
}

/// @title L2ContractsManager_Upgrade_InteropFlagDisabled_Test
/// @notice Tests that interop predeploy upgrades are skipped when the INTEROP sys feature is disabled.
contract L2ContractsManager_Upgrade_InteropFlagDisabled_Test is L2ContractsManager_Upgrade_Test {
    using stdStorage for StdStorage;

    /// @notice The list of interop predeploy addresses.
    address[] internal interopPredeploys;

    function setUp() public override {
        super.setUp();
        skipIfDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);
        interopPredeploys.push(Predeploys.CROSS_L2_INBOX);
        interopPredeploys.push(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER);
        interopPredeploys.push(Predeploys.SUPERCHAIN_ETH_BRIDGE);
        interopPredeploys.push(Predeploys.ETH_LIQUIDITY);
    }

    /// @notice Tests that all 4 interop predeploys retain pre-upgrade implementations when OPTIMISM_PORTAL_INTEROP flag
    /// is disabled.
    function test_upgradeSkipsInteropPredeploys_whenInteropFlagDisabled_succeeds() public {
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

/// @title L2ContractsManager_Upgrade_FeatureFlagMismatch_Test
/// @notice Tests that _loadFullConfig reverts when the INTEROP system customization is enabled
///         but the OPTIMISM_PORTAL_INTEROP dev feature is not.
contract L2ContractsManager_Upgrade_FeatureFlagMismatch_Test is L2ContractsManager_Upgrade_Test {
    using stdStorage for StdStorage;

    /// @notice Tests that upgrade succeeds when the dev feature is enabled but the system feature is not.
    ///         This is a valid transitional state: interop code is deployed but not yet activated.
    function test_upgrade_devEnabledSysDisabled_succeeds() public {
        skipIfDevFeatureDisabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Clear the INTEROP system feature on L1Block while the dev feature remains enabled.
        stdstore.target(Predeploys.L1_BLOCK_ATTRIBUTES).sig("isFeatureEnabled(bytes32)").with_key(Features.INTEROP)
            .checked_write(false);

        _executeUpgrade();
    }

    /// @notice Tests that upgrade reverts when the system feature is enabled but the dev feature is not.
    function test_upgrade_sysEnabledDevDisabled_reverts() public {
        skipIfDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);

        // Set the INTEROP system feature on L1Block while the dev feature remains disabled.
        stdstore.target(Predeploys.L1_BLOCK_ATTRIBUTES).sig("isFeatureEnabled(bytes32)").with_key(Features.INTEROP)
            .checked_write(true);

        vm.expectRevert(L2ContractsManager.L2ContractsManager_FeatureFlagMismatch.selector);
        _executeUpgrade();
    }
}

/// @title L2ContractsManager_Upgrade_Coverage_Test
/// @notice Test that verifies all predeploys receive upgrade calls during L2CM upgrade.
///         Uses Predeploys.sol as the source of truth for which predeploys should be upgraded.
contract L2ContractsManager_Upgrade_Coverage_Test is L2ContractsManager_Upgrade_Test {
    function setUp() public override {
        super.setUp();
        // Skip interop since it requires to call useInterop() on setUp.
        // The upgrade test for interop is in L2ContractsManager_Upgrade_InteropFlagEnabled_Test.
        skipIfDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP);
    }

    /// @notice Checks if a predeploy is an interop predeploy gated behind the OPTIMISM_PORTAL_INTEROP dev feature flag.
    function _isInteropPredeploy(address _predeploy) internal pure returns (bool) {
        return _predeploy == Predeploys.CROSS_L2_INBOX || _predeploy == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER
            || _predeploy == Predeploys.SUPERCHAIN_ETH_BRIDGE || _predeploy == Predeploys.ETH_LIQUIDITY;
    }

    /// @notice Returns CGT-only predeploys that require initialization.
    /// @dev These are separate because they're only deployed on CGT networks.
    function _getCGTInitializablePredeploys() internal pure returns (address[] memory predeploys_) {
        predeploys_ = new address[](1);
        predeploys_[0] = Predeploys.LIQUIDITY_CONTROLLER;
    }

    /// @notice Returns true if a predeploy is a feature predeploy and is disabled.
    /// @param _predeploy The predeploy to check.
    /// @return True if the predeploy is a feature predeploy and feature is disabled, false otherwise.
    function _isFeaturePredeployAndDisabled(address _predeploy) internal view returns (bool) {
        if (!isSysFeatureEnabled(Features.CUSTOM_GAS_TOKEN)) {
            if (_predeploy == Predeploys.NATIVE_ASSET_LIQUIDITY || _predeploy == Predeploys.LIQUIDITY_CONTROLLER) {
                return true;
            }
        }
        if (!isDevFeatureEnabled(DevFeatures.OPTIMISM_PORTAL_INTEROP) && _isInteropPredeploy(_predeploy)) {
            return true;
        }
        return false;
    }

    /// @notice Tests that all predeploys from the registry receive the expected upgrade call.
    ///         Uses vm.expectCall() to verify that upgradeTo or upgradeToAndCall is called.
    /// @dev If L2CM misses a predeploy that exists in PredeployRegistry, this test will fail.
    function test_allPredeploysReceiveUpgradeCall_succeeds() public {
        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();

        for (uint256 i = 0; i < records.length; i++) {
            if (records[i].isVariant) {
                console.log("Skipping variant predeploy", records[i].name);
                continue;
            }
            if (_isFeaturePredeployAndDisabled(records[i].proxy)) {
                console.log("Skipping feature predeploy and feature disabled", records[i].name);
                continue;
            }
            address predeploy = records[i].proxy;

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
    using stdStorage for StdStorage;

    /// @notice Helper function that simulates an existing-chain state where L2DevFeatureFlags
    ///         has not been deployed. Empties the current proxy implementation so that the
    ///         null-safe guard in `_isDevFeatureEnabled` fires and returns false.
    function _simulateNoFlagsImpl() internal {
        address currentImpl = EIP1967Helper.getImplementation(Predeploys.L2_DEV_FEATURE_FLAGS);
        vm.etch(currentImpl, bytes(""));

        // Clear the INTEROP system feature on L1Block to stay consistent with the dev feature
        // being unavailable, otherwise _loadFullConfig will revert on the mismatch check.
        stdstore.target(Predeploys.L1_BLOCK_ATTRIBUTES).sig("isFeatureEnabled(bytes32)").with_key(Features.INTEROP)
            .checked_write(false);
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

    /// @notice Tests that all interop predeploys retain their pre-upgrade implementations
    ///         when the flags implementation has no code.
    function test_upgrade_skipsInteropPredeploys_succeeds() public {
        address[] memory interopPredeploys = new address[](4);
        interopPredeploys[0] = Predeploys.CROSS_L2_INBOX;
        interopPredeploys[1] = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;
        interopPredeploys[2] = Predeploys.SUPERCHAIN_ETH_BRIDGE;
        interopPredeploys[3] = Predeploys.ETH_LIQUIDITY;

        address[] memory preUpgradeImpls = new address[](4);
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
            _findImplByName("L2CrossDomainMessenger"),
            "L2CrossDomainMessenger should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.GAS_PRICE_ORACLE),
            _findImplByName("GasPriceOracle"),
            "GasPriceOracle should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L1_BLOCK_ATTRIBUTES),
            Config.sysFeatureCustomGasToken() ? _findImplByName("L1BlockCGT") : _findImplByName("L1Block"),
            "L1Block should be upgraded"
        );
        assertEq(
            EIP1967Helper.getImplementation(Predeploys.L2_STANDARD_BRIDGE),
            _findImplByName("L2StandardBridge"),
            "L2StandardBridge should be upgraded"
        );
    }
}

/// @title L2ContractsManager_Reverter_Harness
/// @notice Test helper whose runtime bytecode is etched over each initializable predeploy's new
///         implementation in the atomicity test. Exposes `version()` so it passes the L2CM
///         downgrade guard, then reverts from its fallback when the initializer is invoked via
///         `upgradeToAndCall`.
contract L2ContractsManager_Reverter_Harness {
    /// @notice Thrown from the fallback — i.e. from any call that is not `version()`.
    error L2ContractsManager_Reverter_Harness_AlwaysReverts();

    /// @notice Returns a version high enough to pass L2CM's downgrade check against any real
    ///         predeploy version.
    function version() external pure returns (string memory) {
        return "99.0.0";
    }

    /// @notice Reverts on any call that is not `version()` — including the initializer that L2CM
    ///         dispatches via upgradeToAndCall.
    fallback() external payable {
        revert L2ContractsManager_Reverter_Harness_AlwaysReverts();
    }
}

/// @title L2ContractsManager_Upgrade_Atomicity_Test
/// @notice Regression guard: ensures any per-predeploy upgrade failure in
///         `_apply()` aborts the whole upgrade, covering both the `upgradeToAndCall` and
///         `upgradeTo` paths.
contract L2ContractsManager_Upgrade_Atomicity_Test is L2ContractsManager_Upgrade_Test {
    function _countUpgradeablePredeploys(bool _initializable) internal view returns (uint256 count_) {
        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();
        for (uint256 i; i < records.length; i++) {
            if (
                _requiresInitialization(records[i].proxy) == _initializable && _isPredeployUpgradeable(records[i].proxy)
            ) {
                count_++;
            }
        }
    }

    // TODO(#19260): Refactor this when we have a proper single source of truth for the predeploys.
    /// @dev Reverts when `_predeploy` is unmapped so new predeploys cannot slip past this test
    ///      without the helper being extended.
    function _getTargetImpl(address _predeploy) internal view returns (address) {
        // Initializable predeploys (upgradeToAndCall path).
        if (_predeploy == Predeploys.L2_CROSS_DOMAIN_MESSENGER) return _findImplByName("L2CrossDomainMessenger");
        if (_predeploy == Predeploys.L2_STANDARD_BRIDGE) return _findImplByName("L2StandardBridge");
        if (_predeploy == Predeploys.L2_ERC721_BRIDGE) return _findImplByName("L2ERC721Bridge");
        if (_predeploy == Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY) {
            return _findImplByName("OptimismMintableERC20Factory");
        }
        if (_predeploy == Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY) {
            return _findImplByName("OptimismMintableERC721Factory");
        }
        if (_predeploy == Predeploys.SEQUENCER_FEE_WALLET) return _findImplByName("SequencerFeeVault");
        if (_predeploy == Predeploys.BASE_FEE_VAULT) return _findImplByName("BaseFeeVault");
        if (_predeploy == Predeploys.L1_FEE_VAULT) return _findImplByName("L1FeeVault");
        if (_predeploy == Predeploys.OPERATOR_FEE_VAULT) return _findImplByName("OperatorFeeVault");
        if (_predeploy == Predeploys.LIQUIDITY_CONTROLLER) return _findImplByName("LiquidityController");

        // Non-initializable predeploys (upgradeTo path).
        if (_predeploy == Predeploys.GAS_PRICE_ORACLE) return _findImplByName("GasPriceOracle");
        if (_predeploy == Predeploys.L1_BLOCK_ATTRIBUTES) {
            return Config.sysFeatureCustomGasToken() ? _findImplByName("L1BlockCGT") : _findImplByName("L1Block");
        }
        if (_predeploy == Predeploys.L2_TO_L1_MESSAGE_PASSER) {
            return Config.sysFeatureCustomGasToken()
                ? _findImplByName("L2ToL1MessagePasserCGT")
                : _findImplByName("L2ToL1MessagePasser");
        }
        if (_predeploy == Predeploys.PROXY_ADMIN) return _findImplByName("L2ProxyAdmin");
        if (_predeploy == Predeploys.L2_DEV_FEATURE_FLAGS) return _findImplByName("L2DevFeatureFlags");
        if (_predeploy == Predeploys.NATIVE_ASSET_LIQUIDITY) return _findImplByName("NativeAssetLiquidity");
        if (_predeploy == Predeploys.SCHEMA_REGISTRY) return _findImplByName("SchemaRegistry");
        if (_predeploy == Predeploys.EAS) return _findImplByName("EAS");
        if (_predeploy == Predeploys.CROSS_L2_INBOX) return _findImplByName("CrossL2Inbox");
        if (_predeploy == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) {
            return _findImplByName("L2ToL2CrossDomainMessenger");
        }
        if (_predeploy == Predeploys.SUPERCHAIN_ETH_BRIDGE) return _findImplByName("SuperchainETHBridge");
        if (_predeploy == Predeploys.ETH_LIQUIDITY) return _findImplByName("ETHLiquidity");
        if (_predeploy == Predeploys.CONDITIONAL_DEPLOYER) return _findImplByName("ConditionalDeployer");

        revert("L2ContractsManager_Upgrade_Atomicity_Test: unmapped predeploy");
    }

    /// @notice Forces each initializable predeploy's initializer to revert and asserts the whole
    ///         `upgrade()` reverts. The inner `L2ContractsManager_Reverter_Harness_AlwaysReverts` is swallowed
    ///         by `Proxy.upgradeToAndCall` (which wraps the delegatecall result in a `require`);
    ///         the outer revert is therefore the Proxy's string error.
    function test_upgrade_initializerRevertPropagates_reverts() public {
        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();
        uint256 coveredCount;

        for (uint256 i = 0; i < records.length; i++) {
            if (!_requiresInitialization(records[i].proxy)) continue;
            if (!_isPredeployUpgradeable(records[i].proxy)) continue;

            uint256 snapshotId = vm.snapshotState();
            vm.etch(_getTargetImpl(records[i].proxy), address(new L2ContractsManager_Reverter_Harness()).code);

            // Proxy.upgradeToAndCall wraps the inner revert in `require(success, "Proxy: ...")`,
            // so the outer revert is Error(string) with the Proxy's message.
            vm.expectRevert("Proxy: delegatecall to new implementation contract failed");
            _executeUpgrade();

            vm.revertToState(snapshotId);
            coveredCount++;
        }

        assertEq(coveredCount, _countUpgradeablePredeploys(true));
    }

    /// @notice Forces each non-initializable predeploy's new implementation to be code-less and
    ///         asserts the whole `upgrade()` reverts with `L2ContractsManager_EmptyImplementation`.
    ///         Mirrors the initializer test for the `upgradeTo` path.
    function test_upgrade_emptyImplementationPropagates_reverts() public {
        Predeploys.PredeployRecord[] memory records = Predeploys.getUpgradeableRecords();
        uint256 coveredCount;

        for (uint256 i = 0; i < records.length; i++) {
            if (_requiresInitialization(records[i].proxy)) continue;
            if (!_isPredeployUpgradeable(records[i].proxy)) continue;

            uint256 snapshotId = vm.snapshotState();
            address targetImpl = _getTargetImpl(records[i].proxy);
            vm.etch(targetImpl, hex"");

            vm.expectRevert(
                abi.encodeWithSelector(
                    L2ContractsManagerUtils.L2ContractsManager_EmptyImplementation.selector, targetImpl
                )
            );
            _executeUpgrade();

            vm.revertToState(snapshotId);
            coveredCount++;
        }

        assertEq(coveredCount, _countUpgradeablePredeploys(false));
    }
}
