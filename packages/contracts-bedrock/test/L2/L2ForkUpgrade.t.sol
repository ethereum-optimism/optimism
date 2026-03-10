// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Types } from "src/libraries/Types.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

/// @title L2ForkUpgrade_Test
/// @notice Integration test for L2 fork upgrades using NUT bundles.
///         Tests the complete workflow: Fork → Execute → Verify.
contract L2ForkUpgrade_Test is CommonTest {
    /// @notice Struct to capture predeploy state for comparison.
    struct PredeployState {
        address predeploy;
        string version;
    }

    /// @notice Script used for bundle execution.
    ExecuteNUTBundle executeScript;

    /// @notice Script used for bundle generation.
    GenerateNUTBundle generateScript;

    /// @notice Struct to capture pre-upgrade state for comparison.
    struct PreUpgradeState {
        // Versions
        PredeployState[] preUpgradePredeploys;
        // Bridge configuration
        address l2CrossDomainMessengerOtherMessenger;
        address l2StandardBridgeOtherBridge;
        address l2ERC721BridgeOtherBridge;
        address mintableERC20FactoryBridge;
        address mintableERC721FactoryBridge;
        uint256 mintableERC721FactoryRemoteChainID;
        // LiquidityController configuration (CGT only)
        address liquidityControllerOwner;
        string liquidityControllerGasPayingTokenName;
        string liquidityControllerGasPayingTokenSymbol;
        // FeeSplitter configuration
        address feeSplitterSharesCalculator;
        // Fee vault configuration
        address sequencerFeeVaultRecipient;
        uint256 sequencerFeeVaultMinWithdrawal;
        Types.WithdrawalNetwork sequencerFeeVaultWithdrawalNetwork;
        address baseFeeVaultRecipient;
        uint256 baseFeeVaultMinWithdrawal;
        Types.WithdrawalNetwork baseFeeVaultWithdrawalNetwork;
        address l1FeeVaultRecipient;
        uint256 l1FeeVaultMinWithdrawal;
        Types.WithdrawalNetwork l1FeeVaultWithdrawalNetwork;
        address operatorFeeVaultRecipient;
        uint256 operatorFeeVaultMinWithdrawal;
        Types.WithdrawalNetwork operatorFeeVaultWithdrawalNetwork;
        // ProxyAdmin ownership
        address proxyAdminOwner;
        // Feature flags
        bool isInteropEnabled;
        bool isCustomGasToken;
    }

    function setUp() public override {
        super.setUp();

        // Skip if not L2 fork test
        skipIfNotL2ForkTest("requires L2 fork");

        // Skip if L2CM dev feature is not enabled
        skipIfDevFeatureDisabled(DevFeatures.L2CM);

        // Initialize scripts
        executeScript = new ExecuteNUTBundle();
        generateScript = new GenerateNUTBundle();
    }

    /// @notice Tests the complete L2 fork upgrade workflow.
    ///         Executes upgrade and verifies all aspects: implementations, versions, and configurations.
    function test_l2ForkUpgrade_completeUpgrade_succeeds() public {
        // Capture pre-upgrade state
        PreUpgradeState memory preState = _capturePreUpgradeState();

        // Generate bundle
        generateScript.run();

        // Execute bundle on forked L2
        executeScript.execute();

        // Verify all aspects of the upgrade
        _verifyAllVersionsUpdated(preState);
        _verifyInitializationState(preState);
    }

    /// @notice Captures the current state before upgrade for comparison.
    function _capturePreUpgradeState() internal view returns (PreUpgradeState memory state_) {
        // Capture feature flags
        state_.isInteropEnabled = forkL2Live.isInteropEnabled();
        state_.isCustomGasToken = forkL2Live.isCustomGasToken();

        // Capture versions
        state_.preUpgradePredeploys = _getPreUpgradePredeploys();

        // Capture bridge configuration
        state_.l2CrossDomainMessengerOtherMessenger =
            address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).OTHER_MESSENGER());
        state_.l2StandardBridgeOtherBridge =
            address(IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).OTHER_BRIDGE());
        state_.l2ERC721BridgeOtherBridge = address(IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).OTHER_BRIDGE());
        state_.mintableERC20FactoryBridge =
            address(IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).BRIDGE());
        state_.mintableERC721FactoryBridge =
            address(IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).BRIDGE());
        state_.mintableERC721FactoryRemoteChainID =
            IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).REMOTE_CHAIN_ID();

        // Capture LiquidityController configuration (only on CGT networks)
        if (state_.isCustomGasToken) {
            ILiquidityController liquidityController = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
            state_.liquidityControllerOwner = liquidityController.owner();
            state_.liquidityControllerGasPayingTokenName = liquidityController.gasPayingTokenName();
            state_.liquidityControllerGasPayingTokenSymbol = liquidityController.gasPayingTokenSymbol();
        }

        // Capture FeeSplitter configuration
        // eip150-safe
        try IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator() returns (
            ISharesCalculator sharesCalculator_
        ) {
            state_.feeSplitterSharesCalculator = address(sharesCalculator_);
        } catch {
            state_.feeSplitterSharesCalculator = address(0);
        }

        // Capture fee vault configuration
        state_.sequencerFeeVaultRecipient = IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).RECIPIENT();
        state_.sequencerFeeVaultMinWithdrawal =
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).MIN_WITHDRAWAL_AMOUNT();
        // eip150-safe
        try IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).WITHDRAWAL_NETWORK() returns (
            Types.WithdrawalNetwork withdrawalNetwork_
        ) {
            state_.sequencerFeeVaultWithdrawalNetwork = withdrawalNetwork_;
        } catch {
            state_.sequencerFeeVaultWithdrawalNetwork = Types.WithdrawalNetwork.L1;
        }

        state_.baseFeeVaultRecipient = IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).RECIPIENT();
        state_.baseFeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // eip150-safe
        try IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).WITHDRAWAL_NETWORK() returns (
            Types.WithdrawalNetwork withdrawalNetwork_
        ) {
            state_.baseFeeVaultWithdrawalNetwork = withdrawalNetwork_;
        } catch {
            state_.baseFeeVaultWithdrawalNetwork = Types.WithdrawalNetwork.L1;
        }

        state_.l1FeeVaultRecipient = IFeeVault(payable(Predeploys.L1_FEE_VAULT)).RECIPIENT();
        state_.l1FeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.L1_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // eip150-safe
        try IFeeVault(payable(Predeploys.L1_FEE_VAULT)).WITHDRAWAL_NETWORK() returns (
            Types.WithdrawalNetwork withdrawalNetwork_
        ) {
            state_.l1FeeVaultWithdrawalNetwork = withdrawalNetwork_;
        } catch {
            state_.l1FeeVaultWithdrawalNetwork = Types.WithdrawalNetwork.L1;
        }

        state_.operatorFeeVaultRecipient = IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).RECIPIENT();
        state_.operatorFeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // eip150-safe
        try IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).WITHDRAWAL_NETWORK() returns (
            Types.WithdrawalNetwork withdrawalNetwork_
        ) {
            state_.operatorFeeVaultWithdrawalNetwork = withdrawalNetwork_;
        } catch {
            state_.operatorFeeVaultWithdrawalNetwork = Types.WithdrawalNetwork.L1;
        }

        // Capture ProxyAdmin ownership
        state_.proxyAdminOwner = IProxyAdmin(Predeploys.PROXY_ADMIN).owner();
    }

    /// @notice Helper to get pre-upgrade predeploy state.
    function _getPreUpgradePredeploys() internal view returns (PredeployState[] memory predeploys_) {
        predeploys_ = new PredeployState[](Predeploys.getUpgradeablePredeploys().length);
        for (uint256 i = 0; i < Predeploys.getUpgradeablePredeploys().length; i++) {
            predeploys_[i].predeploy = Predeploys.getUpgradeablePredeploys()[i];
            predeploys_[i].version = _getVersion(Predeploys.getUpgradeablePredeploys()[i]);
        }
    }

    /// @notice Helper to get version string from a contract. Returns "0.0.0" if not available.
    function _getVersion(address _contract) internal view returns (string memory) {
        try ISemver(_contract).version() returns (string memory ver_) {
            return ver_;
        } catch {
            return "0.0.0";
        }
    }

    /// @notice Verifies that all contract versions were updated.
    function _verifyAllVersionsUpdated(PreUpgradeState memory _preState) internal view {
        uint256 length = _preState.preUpgradePredeploys.length;
        for (uint256 i = 0; i < length; i++) {
            if (!_preState.isCustomGasToken) {
                if (
                    Predeploys.getUpgradeablePredeploys()[i] == Predeploys.NATIVE_ASSET_LIQUIDITY
                        || Predeploys.getUpgradeablePredeploys()[i] == Predeploys.LIQUIDITY_CONTROLLER
                ) {
                    continue;
                }
            }
            string memory newVersion = _getVersion(_preState.preUpgradePredeploys[i].predeploy);
            string memory oldVersion = _preState.preUpgradePredeploys[i].version;
            assertTrue(
                SemverComp.gte(newVersion, oldVersion) && !SemverComp.eq(newVersion, "0.0.0"),
                string.concat(
                    "Predeploy version not updated: ",
                    Predeploys.getName(_preState.preUpgradePredeploys[i].predeploy),
                    " old=",
                    oldVersion,
                    " new=",
                    newVersion
                )
            );
        }
    }

    /// @notice Verifies that bridge configurations were preserved.
    function _verifyBridgeConfigurations(PreUpgradeState memory _preState) internal view {
        // Verify L2CrossDomainMessenger configuration
        assertEq(
            address(ICrossDomainMessenger(Predeploys.L2_CROSS_DOMAIN_MESSENGER).OTHER_MESSENGER()),
            _preState.l2CrossDomainMessengerOtherMessenger,
            "L2CrossDomainMessenger.OTHER_MESSENGER not preserved"
        );

        // Verify L2StandardBridge configuration
        assertEq(
            address(IStandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).OTHER_BRIDGE()),
            _preState.l2StandardBridgeOtherBridge,
            "L2StandardBridge.OTHER_BRIDGE not preserved"
        );

        // Verify L2ERC721Bridge configuration
        assertEq(
            address(IERC721Bridge(Predeploys.L2_ERC721_BRIDGE).OTHER_BRIDGE()),
            _preState.l2ERC721BridgeOtherBridge,
            "L2ERC721Bridge.OTHER_BRIDGE not preserved"
        );
    }

    /// @notice Verifies that fee vault configurations were preserved.
    function _verifyFeeVaultConfigurations(PreUpgradeState memory _preState) internal view {
        // SequencerFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).RECIPIENT(),
            _preState.sequencerFeeVaultRecipient,
            "SequencerFeeVault.RECIPIENT not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).MIN_WITHDRAWAL_AMOUNT(),
            _preState.sequencerFeeVaultMinWithdrawal,
            "SequencerFeeVault.MIN_WITHDRAWAL_AMOUNT not preserved"
        );
        assertEq(
            uint8(IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).WITHDRAWAL_NETWORK()),
            uint8(_preState.sequencerFeeVaultWithdrawalNetwork),
            "SequencerFeeVault.WITHDRAWAL_NETWORK not preserved"
        );

        // BaseFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).RECIPIENT(),
            _preState.baseFeeVaultRecipient,
            "BaseFeeVault.RECIPIENT not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT(),
            _preState.baseFeeVaultMinWithdrawal,
            "BaseFeeVault.MIN_WITHDRAWAL_AMOUNT not preserved"
        );
        assertEq(
            uint8(IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).WITHDRAWAL_NETWORK()),
            uint8(_preState.baseFeeVaultWithdrawalNetwork),
            "BaseFeeVault.WITHDRAWAL_NETWORK not preserved"
        );

        // L1FeeVault
        assertEq(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).RECIPIENT(),
            _preState.l1FeeVaultRecipient,
            "L1FeeVault.RECIPIENT not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT(),
            _preState.l1FeeVaultMinWithdrawal,
            "L1FeeVault.MIN_WITHDRAWAL_AMOUNT not preserved"
        );
        assertEq(
            uint8(IFeeVault(payable(Predeploys.L1_FEE_VAULT)).WITHDRAWAL_NETWORK()),
            uint8(_preState.l1FeeVaultWithdrawalNetwork),
            "L1FeeVault.WITHDRAWAL_NETWORK not preserved"
        );

        // OperatorFeeVault
        assertEq(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).RECIPIENT(),
            _preState.operatorFeeVaultRecipient,
            "OperatorFeeVault.RECIPIENT not preserved"
        );
        assertEq(
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT(),
            _preState.operatorFeeVaultMinWithdrawal,
            "OperatorFeeVault.MIN_WITHDRAWAL_AMOUNT not preserved"
        );
        assertEq(
            uint8(IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).WITHDRAWAL_NETWORK()),
            uint8(_preState.operatorFeeVaultWithdrawalNetwork),
            "OperatorFeeVault.WITHDRAWAL_NETWORK not preserved"
        );
    }

    /// @notice Verifies that factory configurations were preserved.
    function _verifyFactoryConfigurations(PreUpgradeState memory _preState) internal view {
        // Verify OptimismMintableERC20Factory configuration
        assertEq(
            address(IOptimismMintableERC20Factory(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY).BRIDGE()),
            _preState.mintableERC20FactoryBridge,
            "OptimismMintableERC20Factory.BRIDGE not preserved"
        );

        // Verify OptimismMintableERC721Factory configuration
        assertEq(
            address(IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).BRIDGE()),
            _preState.mintableERC721FactoryBridge,
            "OptimismMintableERC721Factory.BRIDGE not preserved"
        );
        assertEq(
            IOptimismMintableERC721Factory(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY).REMOTE_CHAIN_ID(),
            _preState.mintableERC721FactoryRemoteChainID,
            "OptimismMintableERC721Factory.REMOTE_CHAIN_ID not preserved"
        );
    }

    /// @notice Verifies that LiquidityController configuration was preserved.
    function _verifyLiquidityControllerConfiguration(PreUpgradeState memory _preState) internal view {
        if (!_preState.isCustomGasToken) return;

        ILiquidityController liquidityController = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
        assertEq(
            liquidityController.owner(), _preState.liquidityControllerOwner, "LiquidityController.owner not preserved"
        );
        assertEq(
            liquidityController.gasPayingTokenName(),
            _preState.liquidityControllerGasPayingTokenName,
            "LiquidityController.gasPayingTokenName not preserved"
        );
        assertEq(
            liquidityController.gasPayingTokenSymbol(),
            _preState.liquidityControllerGasPayingTokenSymbol,
            "LiquidityController.gasPayingTokenSymbol not preserved"
        );
    }

    /// @notice Verifies that FeeSplitter configuration was preserved.
    function _verifyFeeSplitterConfiguration(PreUpgradeState memory _preState) internal view {
        assertEq(
            address(IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator()),
            _preState.feeSplitterSharesCalculator,
            "FeeSplitter.sharesCalculator not preserved"
        );
    }

    /// @notice Verifies that ProxyAdmin ownership was preserved.
    function _verifyProxyAdminOwnership(PreUpgradeState memory _preState) internal view {
        assertEq(
            IProxyAdmin(Predeploys.PROXY_ADMIN).owner(),
            _preState.proxyAdminOwner,
            "ProxyAdmin ownership should be preserved"
        );
    }

    /// @notice Verifies that all initializable predeploys are properly initialized after upgrade.
    ///         This ensures no predeploy is left in an uninitialized or partially initialized state.
    function _verifyInitializationState(PreUpgradeState memory _preState) internal view {
        // Verify configuration preservation and initialization
        _verifyBridgeConfigurations(_preState);
        _verifyFeeVaultConfigurations(_preState);
        _verifyFactoryConfigurations(_preState);
        _verifyLiquidityControllerConfiguration(_preState);
        _verifyFeeSplitterConfiguration(_preState);
        _verifyProxyAdminOwnership(_preState);

        // OpenZeppelin v4 Initializable contracts - slot varies by contract
        _verifyOZv4Initialization(Predeploys.L2_CROSS_DOMAIN_MESSENGER, bytes32(0), 20, "L2CrossDomainMessenger");
        _verifyOZv4Initialization(Predeploys.L2_STANDARD_BRIDGE, bytes32(0), 0, "L2StandardBridge");
        _verifyOZv4Initialization(Predeploys.L2_ERC721_BRIDGE, bytes32(0), 0, "L2ERC721Bridge");
        _verifyOZv4Initialization(
            Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY, bytes32(0), 0, "OptimismMintableERC20Factory"
        );
        _verifyOZv4Initialization(
            Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY, bytes32(uint256(1)), 0, "OptimismMintableERC721Factory"
        );
        _verifyOZv4Initialization(Predeploys.FEE_SPLITTER, bytes32(0), 0, "FeeSplitter");

        // LiquidityController (only on custom gas token networks)
        if (_preState.isCustomGasToken) {
            _verifyOZv4Initialization(Predeploys.LIQUIDITY_CONTROLLER, bytes32(0), 0, "LiquidityController");
        }

        // OpenZeppelin v5 Initializable contracts - ERC-7201 slot
        _verifyOZv5Initialization(Predeploys.SEQUENCER_FEE_WALLET, "SequencerFeeVault");
        _verifyOZv5Initialization(Predeploys.BASE_FEE_VAULT, "BaseFeeVault");
        _verifyOZv5Initialization(Predeploys.L1_FEE_VAULT, "L1FeeVault");
        _verifyOZv5Initialization(Predeploys.OPERATOR_FEE_VAULT, "OperatorFeeVault");
    }

    /// @notice Helper to verify OpenZeppelin v4 initialization state.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _slot The storage slot where the initialized value is located.
    /// @param _offset The offset (in bytes from the right) of the initializer value in the slot.
    /// @param _name The name of the predeploy for error messages.
    function _verifyOZv4Initialization(
        address _proxy,
        bytes32 _slot,
        uint8 _offset,
        string memory _name
    )
        internal
        view
    {
        bytes32 slotValue = vm.load(_proxy, _slot);
        uint256 slotUint = uint256(slotValue);

        // Extract the initialized byte at the specified offset
        uint8 initializedValue = uint8((slotUint >> (uint256(_offset) * 8)) & 0xFF);

        // The initialized value should be non-zero (typically 1 or higher)
        assertGt(initializedValue, 0, string.concat(_name, " should be initialized (OZ v4)"));

        // Verify _initializing is false (for OZ v4, this is the next byte after _initialized)
        uint8 initializingValue = uint8((slotUint >> (uint256(_offset + 1) * 8)) & 0xFF);
        assertEq(initializingValue, 0, string.concat(_name, " should not be mid-initialization (OZ v4)"));
    }

    /// @notice Helper to verify OpenZeppelin v5 initialization state.
    /// @param _proxy The proxy address of the predeploy.
    /// @param _name The name of the predeploy for error messages.
    function _verifyOZv5Initialization(address _proxy, string memory _name) internal view {
        // OZ v5 uses ERC-7201 namespaced storage
        // Slot: keccak256(abi.encode(uint256(keccak256("openzeppelin.storage.Initializable")) - 1)) &
        // ~bytes32(uint256(0xff))
        bytes32 ozV5Slot = 0xf0c57e16840df040f15088dc2f81fe391c3923bec73e23a9662efc9c229c6a00;
        bytes32 slotValue = vm.load(_proxy, ozV5Slot);
        uint256 slotUint = uint256(slotValue);

        // Extract uint64 _initialized (low 8 bytes)
        uint64 initializedValue = uint64(slotUint & 0xFFFFFFFFFFFFFFFF);

        // The initialized value should be non-zero (typically 1 or higher)
        assertGt(initializedValue, 0, string.concat(_name, " should be initialized (OZ v5)"));

        // Extract bool _initializing (byte offset 8, bits 64..71)
        uint8 initializingValue = uint8((slotUint >> 64) & 0xFF);
        assertEq(initializingValue, 0, string.concat(_name, " should not be mid-initialization (OZ v5)"));
    }
}
