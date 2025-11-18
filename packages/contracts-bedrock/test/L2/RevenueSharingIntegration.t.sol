// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { CommonTest } from "test/setup/CommonTest.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { ISuperchainRevSharesCalculator } from "interfaces/L2/ISuperchainRevSharesCalculator.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { ICrossDomainMessenger } from "interfaces/universal/ICrossDomainMessenger.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @title RevenueSharingIntegration_Test
/// @notice Integration tests for the complete revenue sharing system including
///         FeeSplitter, SuperchainRevSharesCalculator, L1Withdrawer.
contract RevenueSharingIntegration_Test is CommonTest {
    /// @notice Basis points scale from SuperchainRevSharesCalculator
    uint32 internal constant BASIS_POINT_SCALE = 10_000;
    uint32 internal constant GROSS_SHARE_BPS = 250; // 2.5%
    uint32 internal constant NET_SHARE_BPS = 1_500; // 15%
    uint256 internal disbursementInterval;

    event FeesDisbursed(ISharesCalculator.ShareInfo[] shareInfo, uint256 grossRevenue);
    event FeesReceived(address indexed sender, uint256 amount);
    event WithdrawalInitiated(address indexed recipient, uint256 amount);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);

    function setUp() public override {
        // Resolve features and skip whole test suite if custom gas token is enabled
        resolveFeaturesFromEnv();
        skipIfDevFeatureEnabled(DevFeatures.CUSTOM_GAS_TOKEN);

        // Enable revenue sharing before calling parent setUp
        super.enableRevenueShare();
        super.setUp();

        disbursementInterval = feeSplitter.feeDisbursementInterval();
    }

    /// @notice Configure all vaults to withdraw to FeeSplitter on L2
    function _configureVaultsForFeeSplitter() private {
        // Get the ProxyAdmin owner to configure vaults
        address proxyAdminOwner = proxyAdmin.owner();

        // Configure all vaults to withdraw to FeeSplitter on L2
        vm.startPrank(proxyAdminOwner);
        IFeeVault(payable(address(sequencerFeeVault))).setRecipient(address(feeSplitter));
        IFeeVault(payable(address(sequencerFeeVault))).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);
        IFeeVault(payable(address(sequencerFeeVault))).setMinWithdrawalAmount(0);

        IFeeVault(payable(address(baseFeeVault))).setRecipient(address(feeSplitter));
        IFeeVault(payable(address(baseFeeVault))).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);
        IFeeVault(payable(address(baseFeeVault))).setMinWithdrawalAmount(0);

        IFeeVault(payable(address(l1FeeVault))).setRecipient(address(feeSplitter));
        IFeeVault(payable(address(l1FeeVault))).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);
        IFeeVault(payable(address(l1FeeVault))).setMinWithdrawalAmount(0);

        IFeeVault(payable(address(operatorFeeVault))).setRecipient(address(feeSplitter));
        IFeeVault(payable(address(operatorFeeVault))).setWithdrawalNetwork(Types.WithdrawalNetwork.L2);
        IFeeVault(payable(address(operatorFeeVault))).setMinWithdrawalAmount(0);
        vm.stopPrank();
    }

    /// @notice Helper to fund vaults
    function _fundVaults(uint256 _sequencerFees, uint256 _baseFees, uint256 _l1Fees, uint256 _operatorFees) private {
        vm.deal(address(sequencerFeeVault), _sequencerFees);
        vm.deal(address(baseFeeVault), _baseFees);
        vm.deal(address(l1FeeVault), _l1Fees);
        vm.deal(address(operatorFeeVault), _operatorFees);
    }

    /// @notice Helper to assert the state of all accounts in the revenue sharing flow
    /// @param sequencerFeeBalance Expected balance of sequencer fee vault
    /// @param baseFeeBalance Expected balance of base fee vault
    /// @param l1FeeBalance Expected balance of L1 fee vault
    /// @param operatorFeeBalance Expected balance of operator fee vault
    /// @param l1WithdrawerBalance Expected balance of L1Withdrawer
    /// @param chainFeesRecipientBalance Expected balance of ChainFeesRecipient
    function _assertFullFlowState(
        uint256 sequencerFeeBalance,
        uint256 baseFeeBalance,
        uint256 l1FeeBalance,
        uint256 operatorFeeBalance,
        uint256 l1WithdrawerBalance,
        uint256 chainFeesRecipientBalance
    )
        private
        view
    {
        // Assert vault balances
        assertEq(address(sequencerFeeVault).balance, sequencerFeeBalance, "Incorrect sequencer fee vault balance");
        assertEq(address(baseFeeVault).balance, baseFeeBalance, "Incorrect base fee vault balance");
        assertEq(address(l1FeeVault).balance, l1FeeBalance, "Incorrect L1 fee vault balance");
        assertEq(address(operatorFeeVault).balance, operatorFeeBalance, "Incorrect operator fee vault balance");

        // Assert recipient balances
        assertEq(address(l1Withdrawer).balance, l1WithdrawerBalance, "Incorrect L1Withdrawer balance");
        assertEq(address(chainFeesRecipient).balance, chainFeesRecipientBalance, "Incorrect ChainFeesRecipient balance");
    }

    // Full Revenue Sharing Integration Flow Test
    // Vaults: S=Sequencer, B=Base, L=L1, O=Operator
    // RevSharesCalculator recipients: L1Withdrawer (share), ChainFeesRecipient (remainder)
    // Thresholds: L1Withdrawer=10 ETH
    //  _________________________________________________________________________________
    // | Vaults (S/B/L/O) | L1Withdrawer | ChainFeesRec | Notes                          |
    // |================================================================================|
    // | Initial state                                                                   |
    // |------------------|--------------|--------------|--------------------------------|
    // | 0/0/0/0          | 0            | 0            | -                              |
    // |------------------|--------------|--------------|--------------------------------|
    // | 1. Fund vaults: S=10, B=8, L=2, O=5 ETH                                        |
    // |------------------|--------------|--------------|--------------------------------|
    // | 10/8/2/5         | 0            | 0            | -                              |
    // |------------------|--------------|--------------|--------------------------------|
    // | 2. Call feeSplitter.disburseFees()                                             |
    // |    L1Withdrawer receives 3.45 ETH < 10 ETH threshold                           |
    // |------------------|--------------|--------------|--------------------------------|
    // | 0/0/0/0          | 3.45         | 21.55        | Accumulating                   |
    // |------------------|--------------|--------------|--------------------------------|
    // | 3. Fund vaults: S=40, B=30, L=10, O=20 ETH                                     |
    // |------------------|--------------|--------------|--------------------------------|
    // | 40/30/10/20      | 3.45         | 21.55        | -                              |
    // |------------------|--------------|--------------|--------------------------------|
    // | 4. Call feeSplitter.disburseFees()                                             |
    // |    L1Withdrawer balance: 3.45 + 13.5 = 16.95 ETH > 10 ETH threshold           |
    // |    Triggers withdrawal                                                         |
    // |------------------|--------------|--------------|--------------------------------|
    // | 0/0/0/0          | 0            | 108.05       | L2→L1 triggered                |
    // |------------------|--------------|--------------|--------------------------------|
    // | 5. Fund vaults: S=5, B=5, L=90, O=0 ETH (high L1 fees, gross share > net)     |
    // |------------------|--------------|--------------|--------------------------------|
    // | 5/5/90/0         | 0            | 108.05       | -                              |
    // |------------------|--------------|--------------|--------------------------------|
    // | 6. Call feeSplitter.disburseFees()                                             |
    // |    L1Withdrawer receives 2.5 ETH < 10 ETH threshold, accumulates               |
    // |------------------|--------------|--------------|--------------------------------|
    // | 0/0/0/0          | 2.5          | 205.55       | Accumulating                   |
    // |__________________|______________|______________|________________________________|
    function test_revenueSharing_fullFlow_succeeds() public {
        // Configure vaults to withdraw to FeeSplitter
        _configureVaultsForFeeSplitter();

        // Get recipient addresses
        address shareRecipient = superchainRevSharesCalculator.shareRecipient();
        address remainderRecipient = superchainRevSharesCalculator.remainderRecipient();

        // Fund vaults with test amounts
        uint256[4] memory fees;
        fees[0] = 10 ether; // sequencer
        fees[1] = 8 ether; // base
        fees[2] = 2 ether; // l1
        fees[3] = 5 ether; // operator

        // Step 1: Fund vaults with small amounts
        _fundVaults(fees[0], fees[1], fees[2], fees[3]);

        // Step 2: First disbursement - should accumulate in L1Withdrawer
        vm.warp(block.timestamp + disbursementInterval + 1);
        feeSplitter.disburseFees();

        // Calculate expected values: Gross=25, Net=23, Share=max(0.625, 3.45)=3.45
        uint256 expectedShare1 = (23 ether * uint256(NET_SHARE_BPS)) / BASIS_POINT_SCALE; // 3.45 ETH (net > gross)
        uint256 expectedRemainder1 = 25 ether - expectedShare1; // 21.55 ETH

        // Assert state
        // Vaults: 0/0/0/0
        //L1Withdrawer: 3.45
        //ChainFeesRecipient: 21.55
        _assertFullFlowState(0, 0, 0, 0, expectedShare1, expectedRemainder1);

        // Store remainder balance for later comparison
        uint256 remainderAfterFirst = remainderRecipient.balance;

        // Step 3: Fund vaults with larger amounts
        fees[0] = 40 ether; // sequencer
        fees[1] = 30 ether; // base
        fees[2] = 10 ether; // l1
        fees[3] = 20 ether; // operator

        _fundVaults(fees[0], fees[1], fees[2], fees[3]);

        // Calculate expected values: Gross=100, Net=90, Share=max(2.5, 13.5)=13.5
        uint256 expectedShare2 = (90 ether * uint256(NET_SHARE_BPS)) / BASIS_POINT_SCALE; // 13.5 ETH (net > gross)
        uint256 expectedRemainder2 = 100 ether - expectedShare2; // 86.5 ETH
        uint256 expectedTotalWithdrawal = expectedShare1 + expectedShare2; // 16.95 ETH

        // Expect L2→L1 withdrawal since 16.95 ETH > 10 ETH threshold
        vm.expectCall(
            Predeploys.L2_CROSS_DOMAIN_MESSENGER,
            expectedTotalWithdrawal,
            abi.encodeCall(
                ICrossDomainMessenger.sendMessage, (l1Withdrawer.recipient(), hex"", l1Withdrawer.withdrawalGasLimit())
            )
        );

        // Step 4: Second disbursement - should trigger L2→L1 withdrawal
        vm.warp(block.timestamp + disbursementInterval + 1);
        feeSplitter.disburseFees();

        // L2ToL1MessagePasser should hold the withdrawn funds
        assertEq(
            address(l2ToL1MessagePasser).balance, expectedTotalWithdrawal, "L2ToL1MessagePasser should hold 16.95 ETH"
        );

        // Assert state
        // Vaults: 0/0/0/0
        //L1Withdrawer: 3.45
        //ChainFeesRecipient: 21.55
        _assertFullFlowState(0, 0, 0, 0, 0, remainderAfterFirst + expectedRemainder2);

        // Store remainder balance for final comparison
        uint256 remainderAfterSecond = remainderRecipient.balance;

        // Step 5: Fund vaults again with high L1 fees to make gross share > net share
        fees[0] = 5 ether; // sequencer
        fees[1] = 5 ether; // base
        fees[2] = 90 ether; // l1 (high L1 fees)
        fees[3] = 0 ether; // operator

        _fundVaults(fees[0], fees[1], fees[2], fees[3]);

        // Step 6: Third disbursement - gross share should be chosen, no withdrawal triggered

        // Calculate expected values: Gross=100, Net=10, Share=max(2.5, 1.5)=2.5
        uint256 expectedShare3 = (100 ether * uint256(GROSS_SHARE_BPS)) / BASIS_POINT_SCALE; // 2.5 ETH

        vm.warp(block.timestamp + disbursementInterval + 1);
        feeSplitter.disburseFees();

        //L2ToL1MessagePasser should still hold only the previous withdrawal (16.95 ETH)
        // The 2.5 ETH stays in L1Withdrawer as it's below threshold
        assertEq(
            address(l2ToL1MessagePasser).balance,
            expectedTotalWithdrawal,
            "L2ToL1MessagePasser should still hold 16.95 ETH"
        );
        assertEq(shareRecipient.balance, expectedShare3, "L1Withdrawer should have 2.5 ETH");

        // Final assertions: 0/0/0/0 | 2.5 | 205.55 |
        // Total remainder: 21.55 + 86.5 + 97.5 = 205.55 ETH
        uint256 finalRemainder = remainderAfterSecond + (100 ether - expectedShare3);
        _assertFullFlowState(0, 0, 0, 0, expectedShare3, finalRemainder);
    }

    /// @notice Fuzz test for the revenue sharing calculator and disbursement.
    /// @dev Checks max(net, gross) share is chosen and disbursed correctly.
    function testFuzz_revenueSharing_calculator_succeeds(
        uint256 _sequencerFees,
        uint256 _baseFees,
        uint256 _operatorFees,
        uint256 _l1Fees,
        uint256 _sequencerFees2,
        uint256 _baseFees2,
        uint256 _operatorFees2,
        uint256 _l1Fees2
    )
        public
    {
        // Bound inputs to prevent overflow and ensure reasonable test ranges
        _sequencerFees = bound(_sequencerFees, 0, 100 ether);
        _baseFees = bound(_baseFees, 0, 100 ether);
        _operatorFees = bound(_operatorFees, 0, 100 ether);
        _l1Fees = bound(_l1Fees, 0, 100 ether);

        _sequencerFees2 = bound(_sequencerFees2, 0, 100 ether);
        _baseFees2 = bound(_baseFees2, 0, 100 ether);
        _operatorFees2 = bound(_operatorFees2, 0, 100 ether);
        _l1Fees2 = bound(_l1Fees2, 0, 100 ether);

        // Handle case where grossShare == 0
        // It is 0 when it sums up to less than 40 because of the GROSS_SHARE_BPS = 250 and BASIS_POINT_SCALE = 10_000
        if (_l1Fees + _sequencerFees + _baseFees + _operatorFees < 40) {
            vm.expectRevert(ISuperchainRevSharesCalculator.SharesCalculator_ZeroGrossShare.selector);
            superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);
            return;
        }

        // Configure vaults for disbursement
        _configureVaultsForFeeSplitter();

        {
            // Get share info from calculator first
            ISharesCalculator.ShareInfo[] memory shareInfo =
                superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees, _baseFees, _operatorFees, _l1Fees);

            // Calculate expected values
            uint256 grossRevenue = _sequencerFees + _baseFees + _operatorFees + _l1Fees;
            uint256 grossShare = (grossRevenue * uint256(GROSS_SHARE_BPS)) / BASIS_POINT_SCALE;
            uint256 netShare = ((grossRevenue - _l1Fees) * uint256(NET_SHARE_BPS)) / BASIS_POINT_SCALE;
            uint256 expectedShare = grossShare > netShare ? grossShare : netShare;

            // Assert calculator returns correct amounts
            assertEq(shareInfo[0].amount, expectedShare, "Share recipient should get max(grossShare, netShare)");
            assertEq(
                shareInfo[0].recipient,
                superchainRevSharesCalculator.shareRecipient(),
                "Share recipient address incorrect"
            );
            assertEq(shareInfo[1].amount, grossRevenue - expectedShare, "Remainder recipient should get gross - share");
            assertEq(
                shareInfo[1].recipient,
                superchainRevSharesCalculator.remainderRecipient(),
                "Remainder recipient address incorrect"
            );

            // Fund vaults and perform disbursement
            _fundVaults(_sequencerFees, _baseFees, _l1Fees, _operatorFees);

            uint256 l1WithdrawerBalanceBefore = address(l1Withdrawer).balance;
            uint256 remainderRecipientBalanceBefore = address(chainFeesRecipient).balance;
            uint256 totalShareBalanceAfter = l1WithdrawerBalanceBefore + expectedShare;
            bool willTriggerWithdrawal = totalShareBalanceAfter >= l1Withdrawer.minWithdrawalAmount();

            // Disburse fees
            vm.warp(block.timestamp + disbursementInterval + 1);

            if (willTriggerWithdrawal) {
                vm.expectEmit(true, false, false, true);
                emit WithdrawalInitiated(l1Withdrawer.recipient(), totalShareBalanceAfter);
            }

            feeSplitter.disburseFees();

            // Assert balances
            assertEq(
                address(chainFeesRecipient).balance,
                remainderRecipientBalanceBefore + (grossRevenue - expectedShare),
                "Remainder recipient should receive expected remainder"
            );

            if (willTriggerWithdrawal) {
                assertEq(address(l1Withdrawer).balance, 0, "L1Withdrawer should be empty after withdrawal");
                assertEq(
                    address(l2ToL1MessagePasser).balance,
                    totalShareBalanceAfter,
                    "L2ToL1MessagePasser should hold the withdrawn share amount"
                );
            } else {
                assertEq(
                    address(l1Withdrawer).balance,
                    totalShareBalanceAfter,
                    "L1Withdrawer should hold the share amount when below threshold"
                );
            }
        }

        // Assert all vaults are drained after first disbursement
        assertEq(
            address(sequencerFeeVault).balance, 0, "Sequencer fee vault should be drained after first disbursement"
        );
        assertEq(address(baseFeeVault).balance, 0, "Base fee vault should be drained after first disbursement");
        assertEq(address(l1FeeVault).balance, 0, "L1 fee vault should be drained after first disbursement");
        assertEq(address(operatorFeeVault).balance, 0, "Operator fee vault should be drained after first disbursement");

        // ========== SECOND DISBURSEMENT ==========

        // Handle case where grossShare == 0
        // It is 0 when it sums up to less than 40 because of the GROSS_SHARE_BPS = 250 and BASIS_POINT_SCALE = 10_000
        if (_l1Fees2 + _sequencerFees2 + _baseFees2 + _operatorFees2 < 40) {
            vm.expectRevert(ISuperchainRevSharesCalculator.SharesCalculator_ZeroGrossShare.selector);
            superchainRevSharesCalculator.getRecipientsAndAmounts(_sequencerFees2, _baseFees2, _operatorFees2, _l1Fees2);
            return;
        }

        {
            // Get share info from calculator for second disbursement
            ISharesCalculator.ShareInfo[] memory shareInfo2 = superchainRevSharesCalculator.getRecipientsAndAmounts(
                _sequencerFees2, _baseFees2, _operatorFees2, _l1Fees2
            );

            // Calculate expected values for second disbursement
            uint256 grossRevenue2 = _sequencerFees2 + _baseFees2 + _operatorFees2 + _l1Fees2;
            uint256 grossShare2 = (grossRevenue2 * uint256(GROSS_SHARE_BPS)) / BASIS_POINT_SCALE;
            uint256 netShare2 = ((grossRevenue2 - _l1Fees2) * uint256(NET_SHARE_BPS)) / BASIS_POINT_SCALE;
            uint256 expectedShare2 = grossShare2 > netShare2 ? grossShare2 : netShare2;

            // Assert calculator returns correct amounts for second disbursement
            assertEq(
                shareInfo2[0].amount,
                expectedShare2,
                "Share recipient should get max(grossShare, netShare) for second disbursement"
            );
            assertEq(
                shareInfo2[1].amount,
                grossRevenue2 - expectedShare2,
                "Remainder recipient should get gross - share for second disbursement"
            );

            // Fund vaults for second disbursement
            _fundVaults(_sequencerFees2, _baseFees2, _l1Fees2, _operatorFees2);

            uint256 l1WithdrawerBalanceBefore2 = address(l1Withdrawer).balance;
            uint256 remainderRecipientBalanceBefore2 = address(chainFeesRecipient).balance;
            uint256 l2ToL1MessagePasserBalanceBefore2 = address(l2ToL1MessagePasser).balance;
            uint256 totalShareBalanceAfter2 = l1WithdrawerBalanceBefore2 + expectedShare2;
            bool willTriggerWithdrawal2 = totalShareBalanceAfter2 >= l1Withdrawer.minWithdrawalAmount();

            // Disburse fees for second time
            vm.warp(block.timestamp + disbursementInterval + 1);

            if (willTriggerWithdrawal2) {
                vm.expectEmit(true, false, false, true);
                emit WithdrawalInitiated(l1Withdrawer.recipient(), totalShareBalanceAfter2);
            }

            feeSplitter.disburseFees();

            // Assert balances
            assertEq(
                address(chainFeesRecipient).balance,
                remainderRecipientBalanceBefore2 + (grossRevenue2 - expectedShare2),
                "Remainder recipient should receive expected remainder from second disbursement"
            );

            if (willTriggerWithdrawal2) {
                assertEq(address(l1Withdrawer).balance, 0, "L1Withdrawer should be empty after second withdrawal");
                assertEq(
                    address(l2ToL1MessagePasser).balance,
                    l2ToL1MessagePasserBalanceBefore2 + totalShareBalanceAfter2,
                    "L2ToL1MessagePasser should hold the total withdrawn amount after second disbursement"
                );
            } else {
                assertEq(
                    address(l1Withdrawer).balance,
                    totalShareBalanceAfter2,
                    "L1Withdrawer should hold accumulated share amount when below threshold after second disbursement"
                );
            }

            // Assert all vaults are drained after second disbursement
            assertEq(
                address(sequencerFeeVault).balance, 0, "Sequencer fee vault should be drained after second disbursement"
            );
            assertEq(address(baseFeeVault).balance, 0, "Base fee vault should be drained after second disbursement");
            assertEq(address(l1FeeVault).balance, 0, "L1 fee vault should be drained after second disbursement");
            assertEq(
                address(operatorFeeVault).balance, 0, "Operator fee vault should be drained after second disbursement"
            );
        }
    }
}
