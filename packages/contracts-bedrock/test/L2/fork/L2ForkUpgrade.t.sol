// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";
import { Vm } from "forge-std/Vm.sol";
import { console } from "forge-std/console.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";
import { GenerateNUTBundle } from "scripts/upgrade/GenerateNUTBundle.s.sol";
import { UpgradeUtils } from "scripts/libraries/UpgradeUtils.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Constants } from "src/libraries/Constants.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";
import { Types } from "src/libraries/Types.sol";
import { NetworkUpgradeTxns } from "src/libraries/NetworkUpgradeTxns.sol";

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

/// @title L2ForkUpgrade_TestInit
/// @notice Reusable test initialization for L2 fork upgrade tests.
///         Contains setup, helper functions, and verification logic.
contract L2ForkUpgrade_TestInit is CommonTest {
    /// @notice Script used for bundle execution.
    ExecuteNUTBundle executeScript;

    /// @notice Script used for bundle generation.
    GenerateNUTBundle generateScript;

    /// @notice Common state
    CommonState commonState;

    /// @notice Common state
    struct CommonState {
        bool isInteropEnabled;
        bool isCustomGasToken;
    }

    function setUp() public virtual override {
        super.setUp();

        // Skip if not L2 fork test
        skipIfNotL2ForkTest("L2ForkUpgrade: not a fork test");

        // Skip if L2CM dev feature is not enabled
        skipIfDevFeatureDisabled(DevFeatures.L2CM);

        // Initialize scripts
        executeScript = new ExecuteNUTBundle();
        generateScript = new GenerateNUTBundle();

        // Generate bundle
        generateScript.run();

        // Capture feature flags
        commonState.isInteropEnabled = forkL2Live.isInteropEnabled();
        commonState.isCustomGasToken = forkL2Live.isCustomGasToken();
    }

    /// @notice Returns true if a predeploy is a feature predeploy and is disabled.
    /// @param _predeploy The predeploy to check.
    /// @return True if the predeploy is a feature predeploy and feature is disabled, false otherwise.
    function _isFeaturePredeployAndDisabled(address _predeploy) internal view returns (bool) {
        if (!commonState.isCustomGasToken) {
            if (_predeploy == Predeploys.NATIVE_ASSET_LIQUIDITY || _predeploy == Predeploys.LIQUIDITY_CONTROLLER) {
                return true;
            }
        }
        if (!commonState.isInteropEnabled) {
            if (
                _predeploy == Predeploys.CROSS_L2_INBOX || _predeploy == Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER
                    || _predeploy == Predeploys.SUPERCHAIN_ETH_BRIDGE || _predeploy == Predeploys.ETH_LIQUIDITY
                    || _predeploy == Predeploys.OPTIMISM_SUPERCHAIN_ERC20_FACTORY
                    || _predeploy == Predeploys.OPTIMISM_SUPERCHAIN_ERC20_BEACON
                    || _predeploy == Predeploys.SUPERCHAIN_TOKEN_BRIDGE
            ) {
                return true;
            }
        }
        return false;
    }

    /// @notice Helper to get the expected implementation address for a predeploy.
    /// @dev Handles feature-specific implementations (CGT variants).
    /// @param _predeploy The predeploy address.
    /// @param _name The predeploy name.
    /// @return expectedImpl_ The expected implementation address.
    function _getExpectedImplementation(
        address _predeploy,
        string memory _name
    )
        internal
        view
        returns (address expectedImpl_)
    {
        // Handle feature-specific implementations
        if (_predeploy == Predeploys.L1_BLOCK_ATTRIBUTES) {
            // L1Block uses CGT variant on custom gas token networks
            string memory implName = commonState.isCustomGasToken ? "L1BlockCGT" : "L1Block";
            (expectedImpl_,,,) = generateScript.implementationConfigs(implName);
        } else if (_predeploy == Predeploys.L2_TO_L1_MESSAGE_PASSER) {
            // L2ToL1MessagePasser uses CGT variant on custom gas token networks
            string memory implName = commonState.isCustomGasToken ? "L2ToL1MessagePasserCGT" : "L2ToL1MessagePasser";
            (expectedImpl_,,,) = generateScript.implementationConfigs(implName);
        } else {
            // Standard implementation lookup
            (expectedImpl_,,,) = generateScript.implementationConfigs(_name);
        }
    }
}

/// @title L2ForkUpgrade_Versions_Test
/// @notice Tests that all predeploy versions are updated after the L2 fork upgrade.
contract L2ForkUpgrade_Versions_Test is L2ForkUpgrade_TestInit {
    /// @notice Struct to capture predeploy state for comparison.
    struct PredeployState {
        address predeploy;
        string version;
    }

    /// @notice Struct to capture pre-upgrade version state for comparison.
    struct PreUpgradeVersionState {
        // Predeploy versions
        PredeployState[] preUpgradePredeploys;
    }

    /// @notice Tests that all predeploy versions are updated after upgrade.
    function test_l2ForkUpgrade_versionsUpdated_succeeds() public {
        // Capture pre-upgrade version state
        PreUpgradeVersionState memory preState = _capturePreUpgradeVersionState();

        // Execute bundle on forked L2
        executeScript.execute();

        // Verify all versions were updated
        _verifyAllVersionsUpdated(preState);
    }

    /// @notice Captures the version state before upgrade for comparison.
    function _capturePreUpgradeVersionState() internal view returns (PreUpgradeVersionState memory state_) {
        state_.preUpgradePredeploys = _getPreUpgradePredeploys();
    }

    /// @notice Verifies that all contract versions were updated.
    function _verifyAllVersionsUpdated(PreUpgradeVersionState memory _preState) internal view {
        uint256 length = _preState.preUpgradePredeploys.length;
        for (uint256 i = 0; i < length; i++) {
            if (_isFeaturePredeployAndDisabled(_preState.preUpgradePredeploys[i].predeploy)) {
                console.log(
                    "Skipping feature predeploy and disabled: ",
                    Predeploys.getName(_preState.preUpgradePredeploys[i].predeploy)
                );
                console.log("isCustomGasToken: ", commonState.isCustomGasToken);
                console.log("isInteropEnabled: ", commonState.isInteropEnabled);
                continue;
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
}

/// @title L2ForkUpgrade_Initialization_Test
/// @notice Tests that all initialization configurations are preserved after the L2 fork upgrade.
contract L2ForkUpgrade_Initialization_Test is L2ForkUpgrade_TestInit {
    /// @notice Struct to capture pre-upgrade initialization state for comparison.
    struct PreUpgradeInitializationState {
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
    }

    /// @notice Tests that all initialization configurations are preserved after upgrade.
    function test_l2ForkUpgrade_initializationPreserved_succeeds() public {
        // Capture pre-upgrade initialization state
        PreUpgradeInitializationState memory preState = _capturePreUpgradeInitializationState();

        // Execute bundle on forked L2
        executeScript.execute();

        // Verify initialization state was preserved
        _verifyInitializationState(preState);
    }

    /// @notice Captures the initialization state before upgrade for comparison.
    function _capturePreUpgradeInitializationState()
        internal
        view
        returns (PreUpgradeInitializationState memory state_)
    {
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
        if (commonState.isCustomGasToken) {
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

    /// @notice Verifies that all initializable predeploys are properly initialized after upgrade.
    ///         This ensures no predeploy is left in an uninitialized or partially initialized state.
    function _verifyInitializationState(PreUpgradeInitializationState memory _preState) internal view {
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
        if (commonState.isCustomGasToken) {
            _verifyOZv4Initialization(Predeploys.LIQUIDITY_CONTROLLER, bytes32(0), 0, "LiquidityController");
        }

        // OpenZeppelin v5 Initializable contracts - ERC-7201 slot
        _verifyOZv5Initialization(Predeploys.SEQUENCER_FEE_WALLET, "SequencerFeeVault");
        _verifyOZv5Initialization(Predeploys.BASE_FEE_VAULT, "BaseFeeVault");
        _verifyOZv5Initialization(Predeploys.L1_FEE_VAULT, "L1FeeVault");
        _verifyOZv5Initialization(Predeploys.OPERATOR_FEE_VAULT, "OperatorFeeVault");
    }

    /// @notice Verifies that bridge configurations were preserved.
    function _verifyBridgeConfigurations(PreUpgradeInitializationState memory _preState) internal view {
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
    function _verifyFeeVaultConfigurations(PreUpgradeInitializationState memory _preState) internal view {
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
    function _verifyFactoryConfigurations(PreUpgradeInitializationState memory _preState) internal view {
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
    function _verifyLiquidityControllerConfiguration(PreUpgradeInitializationState memory _preState) internal view {
        if (!commonState.isCustomGasToken) return;

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
    function _verifyFeeSplitterConfiguration(PreUpgradeInitializationState memory _preState) internal view {
        assertEq(
            address(IFeeSplitter(payable(Predeploys.FEE_SPLITTER)).sharesCalculator()),
            _preState.feeSplitterSharesCalculator,
            "FeeSplitter.sharesCalculator not preserved"
        );
    }

    /// @notice Verifies that ProxyAdmin ownership was preserved.
    function _verifyProxyAdminOwnership(PreUpgradeInitializationState memory _preState) internal view {
        assertEq(
            IProxyAdmin(Predeploys.PROXY_ADMIN).owner(),
            _preState.proxyAdminOwner,
            "ProxyAdmin ownership should be preserved"
        );
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

/// @title L2ForkUpgrade_Implementations_Test
/// @notice Tests that all predeploy implementations are correctly upgraded.
contract L2ForkUpgrade_Implementations_Test is L2ForkUpgrade_TestInit {
    /// @notice EIP-1967 implementation storage slot.
    bytes32 internal constant IMPLEMENTATION_SLOT = bytes32(uint256(keccak256("eip1967.proxy.implementation")) - 1);

    /// @notice Tests that all predeploy implementations match expected addresses and have code.
    function test_l2ForkUpgrade_implementationsMatch_succeeds() public {
        // Execute upgrade
        executeScript.execute();

        // Get all upgradeable predeploys
        address[] memory predeploys = Predeploys.getUpgradeablePredeploys();

        // Verify each predeploy's implementation
        for (uint256 i = 0; i < predeploys.length; i++) {
            address predeploy = predeploys[i];

            if (_isFeaturePredeployAndDisabled(predeploy)) {
                continue;
            }

            // Get predeploy name
            string memory name = Predeploys.getName(predeploy);

            // Get expected implementation from config
            address expectedImpl = _getExpectedImplementation(predeploy, name);

            // Get actual implementation from proxy
            address actualImpl = address(uint160(uint256(vm.load(predeploy, IMPLEMENTATION_SLOT))));

            // Verify implementation matches
            assertEq(
                actualImpl,
                expectedImpl,
                string.concat("Implementation mismatch for ", name, ": ", vm.toString(predeploy))
            );

            // Verify implementation has code
            assertGt(
                actualImpl.code.length,
                0,
                string.concat("Implementation has no code for ", name, ": ", vm.toString(actualImpl))
            );
        }
    }
}

/// @title L2ForkUpgrade_Events_Test
/// @notice Tests that all predeploy proxies emit the Upgraded event during the L2 fork upgrade.
contract L2ForkUpgrade_Events_Test is L2ForkUpgrade_TestInit {
    /// @notice EIP-1967 Upgraded event topic.
    /// @dev keccak256("Upgraded(address)")
    bytes32 internal constant UPGRADED_EVENT_TOPIC = 0xbc7cd75a20ee27fd9adebab32041f755214dbc6bffa90cc0225b39da2e5c2d3b;

    /// @notice Tests that all predeploy proxies emit the Upgraded event with correct implementation.
    function test_l2ForkUpgrade_upgradeEventsEmitted_succeeds() public {
        // Get StorageSetter implementation to filter out intermediate upgrade events
        (address storageSetterImpl,,,) = generateScript.implementationConfigs("StorageSetter");

        // Start recording logs
        vm.recordLogs();

        // Execute upgrade bundle
        executeScript.execute();

        // Get all recorded logs
        Vm.Log[] memory logs = vm.getRecordedLogs();

        // Get all upgradeable predeploys
        address[] memory predeploys = Predeploys.getUpgradeablePredeploys();

        // Verify each predeploy emitted the Upgraded event
        for (uint256 i = 0; i < predeploys.length; i++) {
            address predeploy = predeploys[i];

            if (_isFeaturePredeployAndDisabled(predeploy)) {
                continue;
            }

            // Get predeploy name
            string memory name = Predeploys.getName(predeploy);

            // Get expected implementation from config
            address expectedImpl = _getExpectedImplementation(predeploy, name);

            // Find the Upgraded event for this predeploy (skip StorageSetter events)
            bool foundEvent = false;
            for (uint256 j = 0; j < logs.length; j++) {
                // Check if this log is an Upgraded event from the current predeploy
                if (
                    logs[j].emitter == predeploy && logs[j].topics.length > 0
                        && logs[j].topics[0] == UPGRADED_EVENT_TOPIC
                ) {
                    // Decode the implementation address from the event
                    address emittedImpl = address(uint160(uint256(logs[j].topics[1])));

                    // Skip StorageSetter upgrade events (intermediate step for initializable contracts)
                    if (emittedImpl == storageSetterImpl) {
                        continue;
                    }

                    foundEvent = true;

                    // Verify the implementation matches expected
                    assertEq(
                        emittedImpl,
                        expectedImpl,
                        string.concat("Upgraded event implementation mismatch for ", name, ": ", vm.toString(predeploy))
                    );

                    break;
                }
            }

            // Verify the event was found
            assertTrue(foundEvent, string.concat("Upgraded event not found for ", name, ": ", vm.toString(predeploy)));
        }
    }
}

/// @title L2ForkUpgrade_GasProfile_Test
/// @notice Gas profiling test that measures actual gas consumption for each transaction in the upgrade bundle.
contract L2ForkUpgrade_GasProfile_Test is L2ForkUpgrade_TestInit {
    /// @notice Gas measurement for a single transaction.
    struct GasMeasurement {
        uint256 index;
        string intent;
        uint64 gasUsed;
        uint64 gasLimit;
        uint64 recommendedLimit;
        uint256 efficiency; // gasUsed * 100 / gasLimit (percentage)
    }

    /// @notice Safety margin multiplier (150% = 1.5x).
    uint256 internal constant SAFETY_MARGIN_MULTIPLIER = 150;
    uint256 internal constant PERCENTAGE_DENOMINATOR = 100;

    /// @notice Tests gas consumption for all transactions and generates a report.
    function test_l2ForkUpgrade_gasProfile_succeeds() public {
        // Read the bundle
        NetworkUpgradeTxns.NetworkUpgradeTxn[] memory txns =
            NetworkUpgradeTxns.readArtifact(Constants.CURRENT_BUNDLE_PATH);

        console.log(repeat("=", 100));
        console.log("GAS PROFILING REPORT");
        console.log(repeat("=", 100));
        console.log("");
        console.log("Total transactions:", txns.length);
        console.log("");

        // Store measurements
        GasMeasurement[] memory measurements = new GasMeasurement[](txns.length);
        uint256 totalGasUsed = 0;
        uint256 totalGasLimit = 0;

        // Execute and measure each transaction
        for (uint256 i = 0; i < txns.length; i++) {
            NetworkUpgradeTxns.NetworkUpgradeTxn memory txn = txns[i];

            // Ensure sender has sufficient balance
            vm.deal(txn.from, 100 ether);

            // Measure gas
            uint256 gasBefore = gasleft();
            vm.prank(txn.from);
            (bool success, bytes memory returnData) = txn.to.call{ gas: txn.gasLimit }(txn.data);
            uint256 gasAfter = gasleft();

            require(
                success,
                string.concat("Transaction failed: ", txn.intent, " - ", UpgradeUtils.getRevertReason(returnData))
            );

            // Calculate gas used (including overhead)
            uint64 gasUsed = uint64(gasBefore - gasAfter);
            uint64 recommendedLimit = uint64((uint256(gasUsed) * SAFETY_MARGIN_MULTIPLIER) / PERCENTAGE_DENOMINATOR);
            uint256 efficiency = (uint256(gasUsed) * 100) / uint256(txn.gasLimit);

            // Store measurement
            measurements[i] = GasMeasurement({
                index: i,
                intent: txn.intent,
                gasUsed: gasUsed,
                gasLimit: txn.gasLimit,
                recommendedLimit: recommendedLimit,
                efficiency: efficiency
            });

            totalGasUsed += gasUsed;
            totalGasLimit += txn.gasLimit;

            // Print individual transaction report
            console.log("[%s/%s] %s", i + 1, txns.length, txn.intent);
            console.log("  Gas Used:         %s", gasUsed);
            console.log("  Current Limit:    %s", txn.gasLimit);
            console.log("  Recommended:      %s (1.5x actual)", recommendedLimit);
            console.log("  Efficiency:       %s%%", efficiency);
            console.log("");
        }

        // Print summary
        console.log(repeat("=", 100));
        console.log("SUMMARY");
        console.log(repeat("=", 100));
        console.log("Total Gas Used:       %s", totalGasUsed);
        console.log("Total Gas Limit:      %s", totalGasLimit);
        if (totalGasLimit > 0) {
            console.log("Overall Efficiency:   %s%%", (totalGasUsed * 100) / totalGasLimit);
        } else {
            console.log("Overall Efficiency:   N/A (no transactions)");
        }
        console.log("");

        // Print transactions that need adjustment (efficiency < 50% or > 90%)
        console.log("TRANSACTIONS NEEDING ADJUSTMENT:");
        console.log(repeat("-", 100));
        bool foundAdjustments = false;
        for (uint256 i = 0; i < measurements.length; i++) {
            if (measurements[i].efficiency < 50 || measurements[i].efficiency > 90) {
                foundAdjustments = true;
                console.log("[%s] %s", measurements[i].index + 1, measurements[i].intent);
                console.log("  Current: %s | Used: %s", measurements[i].gasLimit, measurements[i].gasUsed);
                console.log(
                    "  Recommended: %s | Efficiency: %s%%", measurements[i].recommendedLimit, measurements[i].efficiency
                );
            }
        }
        if (!foundAdjustments) {
            console.log("All transactions have acceptable efficiency (50-90%)");
        }
        console.log(repeat("=", 100));
    }

    /// @notice Helper function to repeat a string.
    /// @param _str The string to repeat.
    /// @param _count The number of times to repeat.
    /// @return repeated_ The repeated string.
    function repeat(string memory _str, uint256 _count) internal pure returns (string memory repeated_) {
        for (uint256 i = 0; i < _count; i++) {
            repeated_ = string.concat(repeated_, _str);
        }
    }
}
