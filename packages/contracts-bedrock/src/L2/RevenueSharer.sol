// SPDX-License-Identifier: MIT
// TODO gk: this has been largely copy pasted from
// https://github.com/base-org/contracts/blob/main/src/revenue-share/FeeDisburser.sol
pragma solidity 0.8.15;

import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";

import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

/**
 * @title RevenueSharer
 * @dev Withdraws funds from system FeeVault contracts,
 * pays a share of revenue to Optimism
 * and sends the remainder to a configurable adddress on L1.
 */
contract RevenueSharer {
    /*//////////////////////////////////////////////////////////////
                            Constants
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev The basis point scale which revenue share splits are denominated in.
     */
    uint32 public constant BASIS_POINT_SCALE = 10_000;
    /**
     * @dev The minimum gas limit for the FeeDisburser withdrawal transaction to L1.
     */
    uint32 public constant WITHDRAWAL_MIN_GAS = 35_000;
    /**
     * @dev The net revenue percentage denominated in basis points that is used in
     *      Optimism revenue share calculation.
     */
    uint256 public constant OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS = 1_500;
    /**
     * @dev The gross revenue percentage denominated in basis points that is used in
     *      Optimism revenue share calculation.
     */
    uint256 public constant OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS = 250;

    /*//////////////////////////////////////////////////////////////
                            Immutables
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev The address of the Optimism wallet that will receive Optimism's revenue share.
     */
    address payable public immutable OPTIMISM_WALLET;
    /**
     * @dev The address of the L1 wallet that will receive the OP chain runner's share of fees.
     */
    address public immutable L1_WALLET;
    /**
     * @dev The minimum amount of time in seconds that must pass between fee disbursals.
     */
    uint256 public immutable FEE_DISBURSEMENT_INTERVAL;

    /*//////////////////////////////////////////////////////////////
                            Variables
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev The timestamp of the last disbursal.
     */
    uint256 public lastDisbursementTime;
    /**
     * @dev Tracks aggregate net fee revenue which is the sum of sequencer and base fees.
     * @dev Explicity tracking Net Revenue is required to seperate L1FeeVault initiated
     *      withdrawals from Net Revenue calculations.
     */
    uint256 public netFeeRevenue;

    /*//////////////////////////////////////////////////////////////
                            Events
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Emitted when fees are disbursed.
     * @param _disbursementTime The time of the disbursement.
     * @param _paidToOptimism The amount of fees disbursed to Optimism.
     * @param _totalFeesDisbursed The total amount of fees disbursed.
     */
    event FeesDisbursed(uint256 _disbursementTime, uint256 _paidToOptimism, uint256 _totalFeesDisbursed);
    /**
     * @dev Emitted when fees are received from FeeVaults.
     * @param _sender The FeeVault that sent the fees.
     * @param _amount The amount of fees received.
     */
    event FeesReceived(address indexed _sender, uint256 _amount);
    /**
     * @dev Emitted when no fees are collected from FeeVaults at time of disbursement.
     */
    event NoFeesCollected();

    /*//////////////////////////////////////////////////////////////
                            Constructor
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Constructor for the FeeDisburser contract which validates and sets immutable variables.
     * @param _optimismWallet The address which receives Optimism's revenue share.
     * @param _l1Wallet The L1 address which receives the remainder of the revenue.
     * @param _feeDisbursementInterval The minimum amount of time in seconds that must pass between fee disbursals.
     */
    constructor(address payable _optimismWallet, address _l1Wallet, uint256 _feeDisbursementInterval) {
        require(_optimismWallet != address(0), "FeeDisburser: OptimismWallet cannot be address(0)");
        require(_l1Wallet != address(0), "FeeDisburser: L1Wallet cannot be address(0)");
        require(
            _feeDisbursementInterval >= 24 hours, "FeeDisburser: FeeDisbursementInterval cannot be less than 24 hours"
        );

        OPTIMISM_WALLET = _optimismWallet;
        L1_WALLET = _l1Wallet;
        FEE_DISBURSEMENT_INTERVAL = _feeDisbursementInterval;
    }

    /*//////////////////////////////////////////////////////////////
                            External Functions
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Withdraws funds from FeeVaults, sends Optimism their revenue share, and withdraws remaining funds to L1.
     * @dev Implements revenue share business logic as follows:
     *          Net Revenue             = sequencer FeeVault fee revenue + base FeeVault fee revenue
     *          Gross Revenue           = Net Revenue + l1 FeeVault fee revenue
     *          Optimism Revenue Share  = Maximum of 15% of Net Revenue and 2.5% of Gross Revenue
     *          L1 Wallet Revenue Share = Gross Revenue - Optimism Revenue Share
     */
    function disburseFees() external virtual {
        require(
            block.timestamp >= lastDisbursementTime + FEE_DISBURSEMENT_INTERVAL,
            "FeeDisburser: Disbursement interval not reached"
        );

        // Sequencer and base FeeVaults will withdraw fees to the FeeDisburser contract mutating netFeeRevenue
        feeVaultWithdrawal(payable(Predeploys.SEQUENCER_FEE_WALLET));
        feeVaultWithdrawal(payable(Predeploys.BASE_FEE_VAULT));

        feeVaultWithdrawal(payable(Predeploys.L1_FEE_VAULT));

        // Gross revenue is the sum of all fees
        uint256 feeBalance = address(this).balance;

        // TODO gk: etFeeRevenue = feeBalance - l1fees . Be clearer to just subtract this off, and better anticipates
        // future improvements
        // where we may be tracking the actual expenditure on L1

        // Stop execution if no fees were collected
        if (feeBalance == 0) {
            emit NoFeesCollected();
            return;
        }

        lastDisbursementTime = block.timestamp;

        // Net revenue is the sum of sequencer fees and base fees
        uint256 optimismNetRevenueShare = netFeeRevenue * OPTIMISM_NET_REVENUE_SHARE_BASIS_POINTS / BASIS_POINT_SCALE;
        netFeeRevenue = 0;

        uint256 optimismGrossRevenueShare = feeBalance * OPTIMISM_GROSS_REVENUE_SHARE_BASIS_POINTS / BASIS_POINT_SCALE;

        // Optimism's revenue share is the maximum of net and gross revenue
        // TODO gk I think the wording can be improved here  // Optimism's revenue share is the maximum of the
        // respective shares of the net and gross revenue
        uint256 optimismRevenueShare = Math.max(optimismNetRevenueShare, optimismGrossRevenueShare);

        // Send Optimism their revenue share on L2
        require(
            SafeCall.send(OPTIMISM_WALLET, gasleft(), optimismRevenueShare),
            "FeeDisburser: Failed to send funds to Optimism"
        );

        // Send remaining funds to L1 wallet on L1
        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: address(this).balance }(
            L1_WALLET, WITHDRAWAL_MIN_GAS, bytes("")
        );
        emit FeesDisbursed(lastDisbursementTime, optimismRevenueShare, feeBalance);
    }

    /**
     * @dev Receives ETH fees withdrawn from L2 FeeVaults.
     * @dev Will revert if ETH is not sent from L2 FeeVaults.
     */
    receive() external payable virtual {
        if (msg.sender == Predeploys.SEQUENCER_FEE_WALLET || msg.sender == Predeploys.BASE_FEE_VAULT) {
            // Adds value received to net fee revenue if the sender is the sequencer or base FeeVault
            netFeeRevenue += msg.value; // TODO GK: be better to check balance before and after each FeeVault.withdraw()
                // and explicitly label each chunk of ETH pulled in. The tracking of the fees from each vault is hard to
                // reason about as is.
        } else if (msg.sender != Predeploys.L1_FEE_VAULT) {
            revert("FeeDisburser: Only FeeVaults can send ETH to FeeDisburser");
        }
        emit FeesReceived(msg.sender, msg.value);
    }

    /*//////////////////////////////////////////////////////////////
                            Internal Functions
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Withdraws fees from a FeeVault.
     * @param _feeVault The address of the FeeVault to withdraw from.
     * @dev Withdrawal will only occur if the given FeeVault's balance is greater than or equal to
     *        the minimum withdrawal amount.
     */
    function feeVaultWithdrawal(address payable _feeVault) internal {
        require(
            FeeVault(_feeVault).WITHDRAWAL_NETWORK() == FeeVault.WithdrawalNetwork.L2,
            "FeeDisburser: FeeVault must withdraw to L2"
        );
        require(
            FeeVault(_feeVault).RECIPIENT() == address(this),
            "FeeDisburser: FeeVault must withdraw to FeeDisburser contract"
        );
        if (_feeVault.balance >= FeeVault(_feeVault).MIN_WITHDRAWAL_AMOUNT()) {
            FeeVault(_feeVault).withdraw();
        }
    }
}
