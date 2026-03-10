// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { CommonTest } from "test/setup/CommonTest.sol";

// Scripts
import { ExecuteNUTBundle } from "scripts/upgrade/ExecuteNUTBundle.s.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { SemverComp } from "src/libraries/SemverComp.sol";

// Interfaces
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { IStandardBridge } from "interfaces/universal/IStandardBridge.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IERC721Bridge } from "interfaces/universal/IERC721Bridge.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

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
        // Fee vault configuration
        address sequencerFeeVaultRecipient;
        uint256 sequencerFeeVaultMinWithdrawal;
        // Types.WithdrawalNetwork sequencerFeeVaultWithdrawalNetwork;
        address baseFeeVaultRecipient;
        uint256 baseFeeVaultMinWithdrawal;
        // Types.WithdrawalNetwork baseFeeVaultWithdrawalNetwork;
        address l1FeeVaultRecipient;
        uint256 l1FeeVaultMinWithdrawal;
        // Types.WithdrawalNetwork l1FeeVaultWithdrawalNetwork;
        address operatorFeeVaultRecipient;
        uint256 operatorFeeVaultMinWithdrawal;
        // Types.WithdrawalNetwork operatorFeeVaultWithdrawalNetwork;
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
    }

    /// @notice Tests the complete L2 fork upgrade workflow.
    ///         Executes upgrade and verifies all aspects: implementations, versions, and configurations.
    function test_l2ForkUpgrade_completeUpgrade_succeeds() public {
        // Capture pre-upgrade state
        PreUpgradeState memory preState = _capturePreUpgradeState();

        // Execute bundle on forked L2
        executeScript.execute();

        // Verify all aspects of the upgrade
        _verifyAllVersionsUpdated(preState);
        _verifyBridgeConfigurations(preState);
        _verifyFeeVaultConfigurations(preState);
        _verifyFactoryConfigurations(preState);
        _verifyProxyAdminOwnership(preState);
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

        // Capture fee vault configuration
        state_.sequencerFeeVaultRecipient = IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).RECIPIENT();
        state_.sequencerFeeVaultMinWithdrawal =
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).MIN_WITHDRAWAL_AMOUNT();
        // state_.sequencerFeeVaultWithdrawalNetwork =
        //     ISequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).WITHDRAWAL_NETWORK();

        state_.baseFeeVaultRecipient = IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).RECIPIENT();
        state_.baseFeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // state_.baseFeeVaultWithdrawalNetwork =
        // IBaseFeeVault(payable(Predeploys.BASE_FEE_VAULT)).WITHDRAWAL_NETWORK();

        state_.l1FeeVaultRecipient = IFeeVault(payable(Predeploys.L1_FEE_VAULT)).RECIPIENT();
        state_.l1FeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.L1_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // state_.l1FeeVaultWithdrawalNetwork = IL1FeeVault(payable(Predeploys.L1_FEE_VAULT)).WITHDRAWAL_NETWORK();

        state_.operatorFeeVaultRecipient = IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).RECIPIENT();
        state_.operatorFeeVaultMinWithdrawal = IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).MIN_WITHDRAWAL_AMOUNT();
        // state_.operatorFeeVaultWithdrawalNetwork =
        // IOperatorFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).WITHDRAWAL_NETWORK();

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
        // assertEq(
        //     uint8(ISequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).WITHDRAWAL_NETWORK()),
        //     uint8(_preState.sequencerFeeVaultWithdrawalNetwork),
        //     "SequencerFeeVault.WITHDRAWAL_NETWORK not preserved"
        // );

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
        // assertEq(
        //     uint8(IBaseFeeVault(payable(Predeploys.BASE_FEE_VAULT)).WITHDRAWAL_NETWORK()),
        //     uint8(_preState.baseFeeVaultWithdrawalNetwork),
        //     "BaseFeeVault.WITHDRAWAL_NETWORK not preserved"
        // );

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
        // assertEq(
        //     uint8(IL1FeeVault(payable(Predeploys.L1_FEE_VAULT)).WITHDRAWAL_NETWORK()),
        //     uint8(_preState.l1FeeVaultWithdrawalNetwork),
        //     "L1FeeVault.WITHDRAWAL_NETWORK not preserved"
        // );

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
        // assertEq(
        //     uint8(IOperatorFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).WITHDRAWAL_NETWORK()),
        //     uint8(_preState.operatorFeeVaultWithdrawalNetwork),
        //     "OperatorFeeVault.WITHDRAWAL_NETWORK not preserved"
        // );
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
    }

    /// @notice Verifies that ProxyAdmin ownership was preserved.
    function _verifyProxyAdminOwnership(PreUpgradeState memory _preState) internal view {
        assertEq(
            IProxyAdmin(Predeploys.PROXY_ADMIN).owner(),
            _preState.proxyAdminOwner,
            "ProxyAdmin ownership should be preserved"
        );
    }
}
