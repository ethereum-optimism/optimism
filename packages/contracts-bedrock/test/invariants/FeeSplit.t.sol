// SPDX-License-Identifier: MIT
pragma solidity ^0.8.15;

import { StdUtils } from "forge-std/StdUtils.sol";
import { Vm } from "forge-std/Vm.sol";
import { CommonTest } from "test/setup/CommonTest.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { IFeeSplitter } from "interfaces/L2/IFeeSplitter.sol";
import { IL1Withdrawer } from "interfaces/L2/IL1Withdrawer.sol";
import { ISuperchainRevSharesCalculator } from "interfaces/L2/ISuperchainRevSharesCalculator.sol";
import { DevFeatures } from "src/libraries/DevFeatures.sol";

/// @notice A struct to keep track of the state when a disburse call fails
struct DisburseFailureState {
    uint256 sequencerFeeVaultBalance;
    uint256 sequencerFeeVaultMinWithdrawalAmount;
    uint256 baseFeeVaultBalance;
    uint256 baseFeeVaultMinWithdrawalAmount;
    uint256 l1FeeVaultBalance;
    uint256 l1FeeVaultMinWithdrawalAmount;
    uint256 operatorFeeVaultBalance;
    uint256 operatorFeeVaultMinWithdrawalAmount;
    uint256 attemptTimestamp;
    bytes reason;
}

/// @title Handler to call the disburseFees function
contract FeeSplitter_Disburser is StdUtils {
    /// @notice Vm instance
    Vm internal vm;

    /// @notice FeeSplitter contract
    IFeeSplitter public feeSplitter;

    IL1Withdrawer public l1Withdrawer;

    /// @notice Flag to track if a disburseFees() call failed
    bool public txFailed;

    /// @notice Keep track of the balances and timestamp of a failed disbursement
    DisburseFailureState internal failureState;

    /// @notice Aggregate of the vault balances disbursed
    uint256 public ghost_grossRevenueDisbursed;

    /// @notice Keep track of the last aggregated fee disbursed
    uint256 public ghost_lastDisbursementAmount;

    /// @notice Keep track of the l1withdrawer should have withdrawn
    bool public l1withdrawerShouldHaveWithdrawn;

    constructor(Vm _vm, IFeeSplitter _feeSplitter, IL1Withdrawer _l1Withdrawer) {
        vm = _vm;
        feeSplitter = _feeSplitter;
        l1Withdrawer = _l1Withdrawer;
    }

    /// @notice Get the failure state (convenience, to keep the struct)
    function getFailureState() external view returns (DisburseFailureState memory failureState_) {
        failureState_ = failureState;
    }

    /// @notice handler for FeeSplitter.disburseFees()
    /// @dev It update important ghost var, in both success and failure cases:
    /// - success: update the overall amount disbursed (ie add the sum of the vaults balances before disbursement)
    /// - failure: update the failure state (all vault balances and the current timestamp)
    function disburse() public {
        uint256 _sequencerFees = address(Predeploys.SEQUENCER_FEE_WALLET).balance;
        uint256 _baseFees = address(Predeploys.BASE_FEE_VAULT).balance;
        uint256 _l1Fees = address(Predeploys.L1_FEE_VAULT).balance;
        uint256 _operatorFees = address(Predeploys.OPERATOR_FEE_VAULT).balance;
        uint256 _aggregateVaultsBalances = _sequencerFees + _baseFees + _l1Fees + _operatorFees;

        uint256 _l1withdrawerBalanceBeforeDisbursement = address(l1Withdrawer).balance;

        try feeSplitter.disburseFees() {
            // reset the fail flags
            txFailed = false;
            delete failureState;

            // Check if the l1withdrawer should have been triggered and empty its balance
            uint256 _amountToL1Withdrawer = feeSplitter.sharesCalculator().getRecipientsAndAmounts(
                _sequencerFees, _baseFees, _operatorFees, _l1Fees
            )[0].amount;

            if (
                _l1withdrawerBalanceBeforeDisbursement + _amountToL1Withdrawer
                    >= IL1Withdrawer(payable(l1Withdrawer)).minWithdrawalAmount()
            ) {
                l1withdrawerShouldHaveWithdrawn = true;
            } else {
                l1withdrawerShouldHaveWithdrawn = false;
            }

            ghost_grossRevenueDisbursed += _aggregateVaultsBalances;
        } catch (bytes memory _reason) {
            // keep track of the failing state
            txFailed = true;
            failureState.sequencerFeeVaultBalance = address(Predeploys.SEQUENCER_FEE_WALLET).balance;
            failureState.sequencerFeeVaultMinWithdrawalAmount =
                IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).minWithdrawalAmount();
            failureState.baseFeeVaultBalance = address(Predeploys.BASE_FEE_VAULT).balance;
            failureState.baseFeeVaultMinWithdrawalAmount =
                IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).minWithdrawalAmount();
            failureState.l1FeeVaultBalance = address(Predeploys.L1_FEE_VAULT).balance;
            failureState.l1FeeVaultMinWithdrawalAmount =
                IFeeVault(payable(Predeploys.L1_FEE_VAULT)).minWithdrawalAmount();
            failureState.operatorFeeVaultBalance = address(Predeploys.OPERATOR_FEE_VAULT).balance;
            failureState.operatorFeeVaultMinWithdrawalAmount =
                IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).minWithdrawalAmount();
            failureState.attemptTimestamp = block.timestamp;
            failureState.reason = _reason;
        }
    }
}

/// @title Handler to set arbitrary preconditions (balance and block timestamp)
contract FeeSplitter_Preconditions is CommonTest {
    /// @notice modify the min amount to withdraw from a vault
    /// @dev We include the case where min amount is 0 (ie no minimum to withdraw)
    /// @param _minAmount The seed of the min amount to withdraw from a vault
    /// @param _vaultIndex The seed of the vault's index to set the min amount to withdraw from
    function setMinAmount(uint256 _minAmount, uint256 _vaultIndex) public {
        _vaultIndex = bound(_vaultIndex, 0, 3);

        vm.prank(IProxyAdmin(Predeploys.PROXY_ADMIN).owner());

        if (_vaultIndex == 0) {
            IFeeVault(payable(Predeploys.SEQUENCER_FEE_WALLET)).setMinWithdrawalAmount(_minAmount);
        } else if (_vaultIndex == 1) {
            IFeeVault(payable(Predeploys.BASE_FEE_VAULT)).setMinWithdrawalAmount(_minAmount);
        } else if (_vaultIndex == 2) {
            IFeeVault(payable(Predeploys.L1_FEE_VAULT)).setMinWithdrawalAmount(_minAmount);
        } else if (_vaultIndex == 3) {
            IFeeVault(payable(Predeploys.OPERATOR_FEE_VAULT)).setMinWithdrawalAmount(_minAmount);
        }
    }

    /// @notice Warp the block timestamp
    /// @param _seconds The seed of the seconds to warp the block timestamp by
    function warp(uint256 _seconds) public {
        _seconds = bound(_seconds, 0, 10 days);
        vm.warp(block.timestamp + _seconds);
    }

    /// @notice Add collected fee to a vault
    /// @param _amount The seed of amount to add to the vault
    /// @param _vaultIndex The seed of the vault's index to add the fee to
    /// @dev The net and gross revenue have an upper bound to avoid overflows in the shares calculator
    function addCollectedFeeToVault(uint256 _amount, uint256 _vaultIndex) public {
        _vaultIndex = bound(_vaultIndex, 0, 3);
        _amount = bound(_amount, 0, 100 ether);

        if (_vaultIndex == 0) {
            vm.deal(address(Predeploys.SEQUENCER_FEE_WALLET), _amount);
        } else if (_vaultIndex == 1) {
            vm.deal(address(Predeploys.BASE_FEE_VAULT), _amount);
        } else if (_vaultIndex == 2) {
            vm.deal(address(Predeploys.L1_FEE_VAULT), _amount);
        } else if (_vaultIndex == 3) {
            vm.deal(address(Predeploys.OPERATOR_FEE_VAULT), _amount);
        }
    }
}
/// @title Handler to set the call distribution bias for setMinAmount
/// @notice This bias the distribution of calls to setMinAmount, favoring 75% of the calls to set the min amount to 0
/// as it is the "vanilla" case.
/// @dev See https://getfoundry.sh/forge/advanced-testing/invariant-testing#function-call-probability-distribution
/// We favor this over a single wrapper with a seed to branch to keep distinct edges/selectors while using a corpus

contract FeeSplitter_CallDistributionBias is FeeSplitter_Preconditions {
    uint256 public amountZeroCalls;
    uint256 public amountNotZeroCalls;

    function setMinAmountZero1(uint256 _vaultIndex) public {
        amountZeroCalls++;
        setMinAmount(0, _vaultIndex);
    }

    function setMinAmountZero2(uint256 _vaultIndex) public {
        amountZeroCalls++;
        setMinAmount(0, _vaultIndex);
    }

    function setMinAmountZero3(uint256 _vaultIndex) public {
        amountZeroCalls++;
        setMinAmount(0, _vaultIndex);
    }

    function setMinAmountNotZero(uint256 _minAmount, uint256 _vaultIndex) public {
        amountNotZeroCalls++;
        setMinAmount(_minAmount, _vaultIndex);
    }
}

/// @title Invariants for the FeeSplitter
/// @notice The invariants tested are:
/// - no dust accumulation in the FeeSplitter
/// - total disbursed fees should always be equal to the sum of the vault balances before disbursement
/// - disburseFees can only revert if either one of the vault has a balance below it's minimum withdrawal amount
///   or if the disbursement interval has not been reached yet
/// @dev These invariants are covering the system formed by:
/// FeeSplitter, 4 FeeVault's, SuperchainRevSharesCalculator, L1Withdrawer
contract FeeSplitter_Invariant is CommonTest {
    /// @notice Handler for disbursing fees
    FeeSplitter_Disburser public disburser;

    /// @notice Handler to set test preconditions
    FeeSplitter_Preconditions public preconditions;

    /// @notice Handler to set the call distribution bias
    FeeSplitter_CallDistributionBias public callDistributionBias;

    /// @notice Setup: enable the revenue share, deploy handlers and target them.
    function setUp() public override {
        // Resolve features and skip whole test suite if custom gas token is enabled
        resolveFeaturesFromEnv();
        skipIfDevFeatureEnabled(DevFeatures.CUSTOM_GAS_TOKEN);

        super.enableRevenueShare();
        super.setUp();

        disburser = new FeeSplitter_Disburser(vm, feeSplitter, l1Withdrawer);
        preconditions = new FeeSplitter_Preconditions();
        callDistributionBias = new FeeSplitter_CallDistributionBias();

        targetContract(address(disburser));

        targetContract(address(callDistributionBias));
        bytes4[] memory selectors = new bytes4[](4);
        selectors[0] = FeeSplitter_CallDistributionBias.setMinAmountZero1.selector;
        selectors[1] = FeeSplitter_CallDistributionBias.setMinAmountZero2.selector;
        selectors[2] = FeeSplitter_CallDistributionBias.setMinAmountZero3.selector;
        selectors[3] = FeeSplitter_CallDistributionBias.setMinAmountNotZero.selector;
        targetSelector(FuzzSelector({ addr: address(callDistributionBias), selectors: selectors }));

        targetContract(address(preconditions));
        selectors = new bytes4[](2);
        selectors[0] = FeeSplitter_Preconditions.warp.selector;
        selectors[1] = FeeSplitter_Preconditions.addCollectedFeeToVault.selector;
        targetSelector(FuzzSelector({ addr: address(preconditions), selectors: selectors }));
    }

    /// @notice Invariant: The fee splitter balance should always be 0
    /// @dev This invariant doesn't account for direct forced transfers (eg selfdestruct)
    function invariant_noDust() external view {
        assertEq(address(disburser.feeSplitter()).balance, 0);
    }

    /// @notice Invariant: The l1withdrawer should always transfer its whole balance if it reaches the threshold
    function invariant_l1withdrawerWithdrawn() external view {
        if (disburser.l1withdrawerShouldHaveWithdrawn()) {
            assertEq(address(l1Withdrawer).balance, 0);
        }
    }

    /// @notice Invariant: The total disbursed fees should always be equal to the sum of the vault balances before
    /// disbursement
    /// @dev This invariant can also be expressed as "disburseFees is only successful if all vaults can transfer the
    /// fee/all or nothing (0 threshold is accepted)" as, otherwise, some funds would still be in the vaults,
    /// invalidating the equality
    function invariant_balanceConservation() external view {
        assertEq(
            disburser.ghost_grossRevenueDisbursed(),
            address(l1Withdrawer).balance + Predeploys.L2_TO_L1_MESSAGE_PASSER.balance
                + address(chainFeesRecipient).balance
        );
    }

    /// @notice Invariants: these are revert invariants, disburseFees can only revert if either one of the vault
    /// has a balance below it's minimum withdrawal amount (no other revert conditions are possible for the vault)
    /// or if the disbursement interval has not been reached yet (this is making the assumption the recipient are
    /// NOT reverting when receiving the fees).
    /// @dev This invariant is also testing the "no partial disbursement", as the previous one.
    function invariant_disburseReverts() external view {
        if (disburser.txFailed()) {
            DisburseFailureState memory _failureState = disburser.getFailureState();

            uint256 _grossRevenue = _failureState.sequencerFeeVaultBalance + _failureState.baseFeeVaultBalance
                + _failureState.l1FeeVaultBalance + _failureState.operatorFeeVaultBalance;

            // either one of the vaults is below the minimum withdrawal amount
            bool _vaultBelowMinimum = (
                _failureState.sequencerFeeVaultBalance < _failureState.sequencerFeeVaultMinWithdrawalAmount
                    || _failureState.baseFeeVaultBalance < _failureState.baseFeeVaultMinWithdrawalAmount
                    || _failureState.l1FeeVaultBalance < _failureState.l1FeeVaultMinWithdrawalAmount
                    || _failureState.operatorFeeVaultBalance < _failureState.operatorFeeVaultMinWithdrawalAmount
            )
                && keccak256(_failureState.reason)
                    == keccak256(
                        abi.encodeWithSignature(
                            "Error(string)", "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
                        )
                    );

            // not enough time since last disbursement
            bool _tooEarly = _failureState.attemptTimestamp
                < disburser.feeSplitter().lastDisbursementTime() + disburser.feeSplitter().feeDisbursementInterval()
                && bytes4(_failureState.reason) == IFeeSplitter.FeeSplitter_DisbursementIntervalNotReached.selector;

            // no revenue at all
            bool _noRevenue =
                _grossRevenue == 0 && bytes4(_failureState.reason) == IFeeSplitter.FeeSplitter_NoFeesCollected.selector;

            // rounding down error in the shares calculator
            bool _noSharesCalculator = (_grossRevenue * 250) < 10000
                && bytes4(_failureState.reason) == ISuperchainRevSharesCalculator.SharesCalculator_ZeroGrossShare.selector;

            assertTrue(_vaultBelowMinimum || _tooEarly || _noRevenue || _noSharesCalculator);
        }
    }

    /// @notice After invariant: log the call distribution bias
    /// @dev This could be an assertion, but only works significant with big calldepth (or keeping a counter accross all
    /// runs, which needs a file to write to)
    function afterInvariant() external {
        uint256 _totalCalls = callDistributionBias.amountZeroCalls() + callDistributionBias.amountNotZeroCalls();

        if (_totalCalls > 0) {
            emit log_named_uint(
                "% of calls setting the min amount to zero in this run: ",
                callDistributionBias.amountZeroCalls() * 100 / _totalCalls
            );
        }
    }
}
