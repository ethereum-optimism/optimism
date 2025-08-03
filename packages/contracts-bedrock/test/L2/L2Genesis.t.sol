// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Test } from "forge-std/Test.sol";
import { EIP1967Helper } from "test/mocks/EIP1967Helper.sol";
import { L2Genesis } from "scripts/L2Genesis.s.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

import { ISequencerFeeVault } from "interfaces/L2/ISequencerFeeVault.sol";
import { IBaseFeeVault } from "interfaces/L2/IBaseFeeVault.sol";
import { IL1FeeVault } from "interfaces/L2/IL1FeeVault.sol";
import { IOptimismMintableERC20Factory } from "interfaces/universal/IOptimismMintableERC20Factory.sol";
import { IOptimismMintableERC721Factory } from "interfaces/L2/IOptimismMintableERC721Factory.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { IGovernanceToken } from "interfaces/governance/IGovernanceToken.sol";
import { IGasPriceOracle } from "interfaces/L2/IGasPriceOracle.sol";

/// @title L2GenesisTest
/// @notice Test suite for L2Genesis script.
contract L2GenesisTest is Test {
    L2Genesis.Input internal input;

    L2Genesis internal genesis;

    function setUp() public {
        genesis = new L2Genesis();
    }

    function test_run_succeeds() external {
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
            governanceTokenOwner: address(0x0000000000000000000000000000000000000008),
            fork: 7, // Jovian
            useInterop: true,
            enableGovernance: true,
            fundDevAccounts: true
        });
        genesis.run(input);

        testProxyAdmin();
        testPredeploys();
        testVaults();
        testGovernance();
        testFactories();
        testForks();
    }

    function testProxyAdmin() internal view {
        assertEq(input.opChainProxyAdminOwner, IProxyAdmin(Predeploys.PROXY_ADMIN).owner());
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
            if (!Predeploys.isSupportedPredeploy(addr, true)) {
                continue;
            }

            // All proxied predeploys should have the 1967 admin slot set to the ProxyAdmin predeploy
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

        assertEq(baseFeeVault.recipient(), input.baseFeeVaultRecipient);
        assertEq(baseFeeVault.MIN_WITHDRAWAL_AMOUNT(), input.baseFeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(baseFeeVault.WITHDRAWAL_NETWORK()), uint8(input.baseFeeVaultWithdrawalNetwork));

        assertEq(l1FeeVault.recipient(), input.l1FeeVaultRecipient);
        assertEq(l1FeeVault.MIN_WITHDRAWAL_AMOUNT(), input.l1FeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(l1FeeVault.WITHDRAWAL_NETWORK()), uint8(input.l1FeeVaultWithdrawalNetwork));

        assertEq(sequencerFeeVault.recipient(), input.sequencerFeeVaultRecipient);
        assertEq(sequencerFeeVault.MIN_WITHDRAWAL_AMOUNT(), input.sequencerFeeVaultMinimumWithdrawalAmount);
        assertEq(uint8(sequencerFeeVault.WITHDRAWAL_NETWORK()), uint8(input.sequencerFeeVaultWithdrawalNetwork));
    }

    function testGovernance() internal view {
        IGovernanceToken token = IGovernanceToken(payable(Predeploys.GOVERNANCE_TOKEN));
        assertEq(token.owner(), input.governanceTokenOwner);
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
}
