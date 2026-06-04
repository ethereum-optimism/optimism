// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Testing
import { Test } from "test/setup/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";

// Scripts
import { L2Genesis } from "scripts/L2Genesis.s.sol";
import { Fork, LATEST_FORK } from "scripts/libraries/Config.sol";

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";
import { Features } from "src/libraries/Features.sol";

// Interfaces
import { IL1Block } from "interfaces/L2/IL1Block.sol";
import { ISequencerFeeVault } from "interfaces/L2/ISequencerFeeVault.sol";
import { IBaseFeeVault } from "interfaces/L2/IBaseFeeVault.sol";
import { IL1FeeVault } from "interfaces/L2/IL1FeeVault.sol";
import { IOperatorFeeVault } from "interfaces/L2/IOperatorFeeVault.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IL2ProxyAdmin } from "interfaces/L2/IL2ProxyAdmin.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";
import { ILiquidityController } from "interfaces/L2/ILiquidityController.sol";
import { INativeAssetLiquidity } from "interfaces/L2/INativeAssetLiquidity.sol";

/// @title L2Genesis_TestInit
/// @notice Reusable test initialization for `L2Genesis` tests.
abstract contract L2Genesis_TestInit is Test {
    L2Genesis.Input internal input;

    L2Genesis internal genesis;

    function setUp() public virtual {
        genesis = new L2Genesis();
    }

    function testProxyAdmin() internal view {
        // Verify owner in the proxy
        assertEq(input.opChainProxyAdminOwner, IL2ProxyAdmin(Predeploys.PROXY_ADMIN).owner());

        // Verify owner in the implementation to catch storage shifting issues
        // The implementation is stored in the code namespace
        address proxyAdminImpl = Predeploys.predeployToCodeNamespace(Predeploys.PROXY_ADMIN);

        assertEq(
            IL2ProxyAdmin(proxyAdminImpl).owner(), address(0), "ProxyAdmin implementation owner should match expected"
        );
    }

    function testPredeploys() internal view {
        uint160 prefix = uint160(0x420) << 148;

        for (uint256 i = 0; i < Predeploys.PREDEPLOY_COUNT; i++) {
            address addr = address(prefix | uint160(i));
            // If it's not proxied, skip next checks.
            if (Predeploys.notProxied(addr)) {
                continue;
            }

            // All predeploys should have code
            assertGt(addr.code.length, 0);
            // All proxied predeploys should have the 1967 admin slot set to the ProxyAdmin
            assertEq(Predeploys.PROXY_ADMIN, EIP1967Helper.getAdmin(addr));

            // If it's not a supported predeploy, skip next checks.
            if (
                !Predeploys.isSupportedPredeploy(
                    addr, uint256(LATEST_FORK), input.useCustomGasToken, input.useInterop, input.devFeatureBitmap
                )
            ) {
                continue;
            }

            // All proxied predeploys should have the 1967 admin slot set to the ProxyAdmin
            // predeploy
            address impl = Predeploys.predeployToCodeNamespace(addr);
            assertGt(impl.code.length, 0);
        }

        assertGt(Predeploys.WETH.code.length, 0);
        assertGt(Predeploys.GOVERNANCE_TOKEN.code.length, 0);
    }

    function testVaults() internal view {
        IBaseFeeVault baseFeeVault = IBaseFeeVault(payable(Predeploys.BASE_FEE_VAULT));
        IL1FeeVault l1FeeVault = IL1FeeVault(payable(Predeploys.L1_FEE_VAULT));
        ISequencerFeeVault sequencerFeeVault = ISequencerFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET));
        IOperatorFeeVault operatorFeeVault = IOperatorFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT));

        assertEq(baseFeeVault.RECIPIENT(), input.baseFeeVaultRecipient);
        assertEq(baseFeeVault.recipient(), input.baseFeeVaultRecipient);
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), input.baseFeeVaultMinimumWithdrawalAmount);
        assertEq(baseFeeVault.minWithdrawalAmount(), input.baseFeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(baseFeeVault.WITHDRAWAL_NETWORK()), uint8(input.baseFeeVaultWithdrawalNetwork));
        assertEq(uint8(baseFeeVault.withdrawalNetwork()), uint8(input.baseFeeVaultWithdrawalNetwork));

        assertEq(l1FeeVault.RECIPIENT(), input.l1FeeVaultRecipient);
        assertEq(l1FeeVault.recipient(), input.l1FeeVaultRecipient);
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), input.l1FeeVaultMinimumWithdrawalAmount);
        assertEq(l1FeeVault.minWithdrawalAmount(), input.l1FeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(l1FeeVault.WITHDRAWAL_NETWORK()), uint8(input.l1FeeVaultWithdrawalNetwork));
        assertEq(uint8(l1FeeVault.withdrawalNetwork()), uint8(input.l1FeeVaultWithdrawalNetwork));

        assertEq(sequencerFeeVault.RECIPIENT(), input.sequencerFeeVaultRecipient);
        assertEq(sequencerFeeVault.recipient(), input.sequencerFeeVaultRecipient);
        assertEq(sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT(), input.sequencerFeeVaultMinimumWithdrawalAmount);
        assertEq(sequencerFeeVault.minWithdrawalAmount(), input.sequencerFeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(sequencerFeeVault.WITHDRAWAL_NETWORK()), uint8(input.sequencerFeeVaultWithdrawalNetwork));
        assertEq(uint8(sequencerFeeVault.withdrawalNetwork()), uint8(input.sequencerFeeVaultWithdrawalNetwork));

        assertEq(operatorFeeVault.RECIPIENT(), input.operatorFeeVaultRecipient);
        assertEq(operatorFeeVault.recipient(), input.operatorFeeVaultRecipient);
        assertEq(operatorFeeVault.MIN_WITHDRAWAL_AMOUNT(), input.operatorFeeVaultMinimumWithdrawalAmount);
        assertEq(operatorFeeVault.minWithdrawalAmount(), input.operatorFeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(operatorFeeVault.WITHDRAWAL_NETWORK()), uint8(input.operatorFeeVaultWithdrawalNetwork));
        assertEq(uint8(operatorFeeVault.withdrawalNetwork()), uint8(input.operatorFeeVaultWithdrawalNetwork));
    }

    function testGovernance() internal view {
        IGovernanceToken token = IGovernanceToken(payable(Predeploys.GOVERNANCE_TOKEN));

        // Verify owner (existing check)
        assertEq(token.owner(), input.governanceTokenOwner);

        // Verify name and symbol to catch storage shifting issues
        // These should match the values hardcoded in GovernanceToken constructor
        assertEq(token.name(), "Optimism", "GovernanceToken name should be 'Optimism'");
        assertEq(token.symbol(), "OP", "GovernanceToken symbol should be 'OP'");
    }

    function testFactories() internal view {
        IOptimismMintableERC20Factory erc20Factory =
            IOptimismMintableERC20Factory(payable(Predeploys.OPTIMISM_MINTABLE_ERC20_FACTORY));
        IOptimismMintableERC721Factory erc721Factory =
            IOptimismMintableERC721Factory(payable(Predeploys.OPTIMISM_MINTABLE_ERC721_FACTORY));

        assertEq(erc20Factory.bridge(), Predeploys.L2_STANDARD_BRIDGE);
        assertEq(erc721Factory.bridge(), Predeploys.L2_ERC721_BRIDGE);
        assertEq(erc721Factory.remoteChainID(), input.l1ChainID);
    }

    function testForks() internal view {
        // The fork should be set to Isthmus at least. Check by validating the GasPriceOracle
        IGasPriceOracle gasPriceOracle = IGasPriceOracle(payable(Predeploys.GAS_PRICE_ORACLE));
        assertEq(gasPriceOracle.isEcotone(), true);
        assertEq(gasPriceOracle.isFjord(), true);
        assertEq(gasPriceOracle.isIsthmus(), true);
    }

    function testCGT() internal view {
        // Test LiquidityController deployment
        ILiquidityController controller = ILiquidityController(Predeploys.LIQUIDITY_CONTROLLER);
        assertEq(controller.owner(), input.liquidityControllerOwner);
        assertEq(controller.gasPayingTokenName(), input.gasPayingTokenName);
        assertEq(controller.gasPayingTokenSymbol(), input.gasPayingTokenSymbol);

        // Test NativeAssetLiquidity deployment and funding
        INativeAssetLiquidity liquidity = INativeAssetLiquidity(Predeploys.NATIVE_ASSET_LIQUIDITY);
        assertEq(address(liquidity).balance, type(uint248).max);

        // Verify predeploys have code
        assertGt(Predeploys.LIQUIDITY_CONTROLLER.code.length, 0);
        assertGt(Predeploys.NATIVE_ASSET_LIQUIDITY.code.length, 0);
    }
}

/// @title L2Genesis_Run_Test
/// @notice Tests the `run` function of the `L2Genesis` contract.
contract L2Genesis_Run_Test is L2Genesis_TestInit {
    function setUp() public override {
        super.setUp();
        // Set up default input configuration
        input = L2Genesis.Input({
            l1ChainID: 1,
            l2ChainID: 2,
            l1CrossDomainMessengerProxy: payable(address(0x0000000000000000000000000000000000000001)),
            l1StandardBridgeProxy: payable(address(0x0000000000000000000000000000000000000002)),
            l1ERC721BridgeProxy: payable(address(0x0000000000000000000000000000000000000003)),
            opChainProxyAdminOwner: address(0x0000000000000000000000000000000000000004),
            sequencerFeeVaultRecipient: address(0x0000000000000000000000000000000000000005),
            sequencerFeeVaultMinimumWithdrawalAmount: 1,
            sequencerFeeVaultWithdrawalNetwork: 1,
            baseFeeVaultRecipient: address(0x0000000000000000000000000000000000000006),
            baseFeeVaultMinimumWithdrawalAmount: 1,
            baseFeeVaultWithdrawalNetwork: 1,
            l1FeeVaultRecipient: address(0x0000000000000000000000000000000000000007),
            l1FeeVaultMinimumWithdrawalAmount: 1,
            l1FeeVaultWithdrawalNetwork: 1,
            operatorFeeVaultRecipient: address(0x0000000000000000000000000000000000000008),
            operatorFeeVaultMinimumWithdrawalAmount: 1,
            operatorFeeVaultWithdrawalNetwork: 1,
            governanceTokenOwner: address(0x0000000000000000000000000000000000000009),
            fork: uint256(LATEST_FORK),
            enableGovernance: true,
            fundDevAccounts: true,
            useCustomGasToken: false,
            useInterop: false,
            gasPayingTokenName: "",
            gasPayingTokenSymbol: "",
            nativeAssetLiquidityAmount: type(uint248).max,
            liquidityControllerOwner: address(0x000000000000000000000000000000000000000d),
            devFeatureBitmap: bytes32(0)
        });
    }

    function test_run_succeeds() external {
        genesis.run(input);

        testProxyAdmin();
        testPredeploys();
        testVaults();
        testGovernance();
        testFactories();
        testForks();
    }

    /// @notice Helper function to configure input for interop enabled tests.
    function _setInputInteropEnabled() internal {
        input.useInterop = true;
        input.devFeatureBitmap = bytes32(DevFeatures.OPTIMISM_PORTAL_INTEROP);
    }

    /// @notice Asserts that the interop predeploys are present in genesis.
    function testInterop() internal view {
        assertGt(Predeploys.CROSS_L2_INBOX.code.length, 0, "CrossL2Inbox must have code");
        assertGt(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER.code.length, 0, "L2ToL2CrossDomainMessenger must have code");
    }

    /// @notice Tests that the run function succeeds when interop is enabled.
    function test_run_withInterop_succeeds() external {
        _setInputInteropEnabled();
        genesis.run(input);

        testProxyAdmin();
        testPredeploys();
        testVaults();
        testGovernance();
        testFactories();
        testForks();
        testInterop();
    }

    /// @notice Helper function to configure input for CGT enabled tests.
    function _setInputCGTEnabled() internal {
        input.useCustomGasToken = true;
        input.gasPayingTokenName = "Custom Gas Token";
        input.gasPayingTokenSymbol = "CGT";
    }

    /// @notice Tests that the run function succeeds when CGT is enabled.
    /// @dev Tests that LiquidityController and NativeAssetLiquidity are deployed.
    function test_run_cgt_succeeds() external {
        _setInputCGTEnabled();
        genesis.run(input);

        testProxyAdmin();
        testPredeploys();
        testVaults();
        testGovernance();
        testFactories();
        testForks();
        testCGT();
    }

    /// @notice Tests that the run function reverts when CGT is enabled and sequencerFeeVault withdrawal network is L1.
    function test_cgt_sequencerVault_reverts() external {
        _setInputCGTEnabled();
        input.sequencerFeeVaultWithdrawalNetwork = 0;
        vm.expectRevert("FeeVault: withdrawalNetwork type cannot be L1 when custom gas token is enabled");
        genesis.run(input);
    }

    /// @notice Tests that the run function reverts when CGT is enabled and baseFeeVault withdrawal network is L1.
    function test_cgt_baseFeeVault_reverts() external {
        _setInputCGTEnabled();
        input.baseFeeVaultWithdrawalNetwork = 0;
        vm.expectRevert("FeeVault: withdrawalNetwork type cannot be L1 when custom gas token is enabled");
        genesis.run(input);
    }

    /// @notice Tests that the run function reverts when CGT is enabled and l1FeeVault withdrawal network is L1.
    function test_cgt_l1FeeVault_reverts() external {
        _setInputCGTEnabled();
        input.l1FeeVaultWithdrawalNetwork = 0;
        vm.expectRevert("FeeVault: withdrawalNetwork type cannot be L1 when custom gas token is enabled");
        genesis.run(input);
    }

    /// @notice Tests that the run function reverts when nativeAssetLiquidityAmount exceeds type(uint248).max.
    function test_cgt_liquidityAmount_reverts() external {
        _setInputCGTEnabled();
        input.nativeAssetLiquidityAmount = uint256(type(uint248).max) + 1;
        vm.expectRevert("L2Genesis: native asset liquidity amount must be less than or equal to type(uint248).max");
        genesis.run(input);
    }

    /// @notice Tests that enabling l2cm succeeds.
    function test_run_l2cm_succeeds() external {
        input.devFeatureBitmap |= DevFeatures.L2CM;
        genesis.run(input);

        testProxyAdmin();
        testPredeploys();
        testVaults();
        testGovernance();
        testFactories();
        testForks();
    }

    /// @notice Tests that run reverts when useInterop is true but the OPTIMISM_PORTAL_INTEROP dev bit is not set.
    function test_run_useInteropWithoutDevBit_reverts() external {
        input.useInterop = true;
        // devFeatureBitmap left at 0 — OPTIMISM_PORTAL_INTEROP bit not set
        vm.expectRevert("L2Genesis: useInterop and OPTIMISM_PORTAL_INTEROP devFeature bit must agree");
        genesis.run(input);
    }

    /// @notice Tests that run reverts when the OPTIMISM_PORTAL_INTEROP dev bit is set but useInterop is false.
    function test_run_devBitWithoutUseInterop_reverts() external {
        input.useInterop = false;
        input.devFeatureBitmap = bytes32(DevFeatures.OPTIMISM_PORTAL_INTEROP);
        vm.expectRevert("L2Genesis: useInterop and OPTIMISM_PORTAL_INTEROP devFeature bit must agree");
        genesis.run(input);
    }

    /// @notice Tests that when interop is scheduled for a later fork (genesis fork < INTEROP),
    ///         the runtime INTEROP feature flag on L1Block is NOT set at genesis. The flag will
    ///         instead be flipped at the activation block by op-node's setFeature deposit wrapper.
    function test_setL1Block_interopScheduledNotActive_succeeds() external {
        _setInputInteropEnabled();
        input.fork = uint256(Fork.ISTHMUS);
        genesis.run(input);

        assertEq(
            IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isFeatureEnabled(Features.INTEROP),
            false,
            "INTEROP runtime flag must not be set at genesis when fork < INTEROP"
        );
    }

    /// @notice Tests that when a chain is born at or beyond the Interop fork, the runtime INTEROP
    ///         feature flag on L1Block IS set at genesis.
    function test_setL1Block_interopAtGenesis_succeeds() external {
        _setInputInteropEnabled();
        input.fork = uint256(Fork.INTEROP);
        genesis.run(input);

        assertEq(
            IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isFeatureEnabled(Features.INTEROP),
            true,
            "INTEROP runtime flag must be set at genesis when fork >= INTEROP"
        );
    }

    /// @notice Sanity check: with useInterop disabled and fork < INTEROP, the runtime INTEROP flag
    ///         is unset.
    function test_setL1Block_interopDisabled_succeeds() external {
        input.useInterop = false;
        input.fork = uint256(Fork.ISTHMUS);
        genesis.run(input);

        assertEq(
            IL1Block(Predeploys.L1_BLOCK_ATTRIBUTES).isFeatureEnabled(Features.INTEROP),
            false,
            "INTEROP runtime flag must not be set when useInterop is false"
        );
    }
}
