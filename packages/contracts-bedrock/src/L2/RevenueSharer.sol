// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";

import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000022
/// @title RevenueSharer
/// @dev Withdraws funds from system FeeVault contracts,
/// pays a share of revenue to a designated Beneficiary
/// and sends the remainder to a configurable adddress on L1.
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
     * @dev The percentage coeffieicnt of revenue denominated in basis points that is used in
     *      Optimism revenue share calculation.
     */
    uint256 public constant REVENUE_COEFFICIENT_BASIS_POINTS = 1_500;
    /**
     * @dev The percentage coefficient of profit denominated in basis points that is used in
     *      Optimism revenue share calculation.
     */
    uint256 public constant PROFIT_COEFFICIENT_BASIS_POINTS = 250;

    /*//////////////////////////////////////////////////////////////
                            Immutables
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev The address of the Optimism wallet that will receive Optimism's revenue share.
     */
    address payable public immutable BENEFICIARY;
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
     * @param _share The amount of fees shared to the Beneficiary.
     * @param _total The total funds distributed.
     */
    event FeesDisbursed(uint256 _disbursementTime, uint256 _share, uint256 _total);
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
     * @param _beneficiary The address which receives the revenue share.
     * @param _l1Wallet The L1 address which receives the remainder of the revenue.
     * @param _feeDisbursementInterval The minimum amount of time in seconds that must pass between fee disbursals.
     */
    constructor(address payable _beneficiary, address _l1Wallet, uint256 _feeDisbursementInterval) {
        require(_beneficiary != address(0), "FeeDisburser: OptimismWallet cannot be address(0)");
        require(_l1Wallet != address(0), "FeeDisburser: L1Wallet cannot be address(0)");
        require(
            _feeDisbursementInterval >= 24 hours, "FeeDisburser: FeeDisbursementInterval cannot be less than 24 hours"
        );

        BENEFICIARY = _beneficiary;
        L1_WALLET = _l1Wallet;
        FEE_DISBURSEMENT_INTERVAL = _feeDisbursementInterval;
    }

    /*//////////////////////////////////////////////////////////////
                            External Functions
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Withdraws funds from FeeVaults, sends Optimism their revenue share, and withdraws remaining funds to L1.
     */
    function execute() external virtual {
        // Pull in revenue
        uint256 d = feeVaultWithdrawal(Predeploys.L1_FEE_VAULT);
        uint256 b = feeVaultWithdrawal(Predeploys.BASE_FEE_VAULT);
        uint256 q = feeVaultWithdrawal(Predeploys.SEQUENCER_FEE_WALLET);

        // Compute expenditure
        uint256 e = getL1FeeExpenditure();

        // Compute revenue and profit
        uint256 r = d + b + q; // revenue
        uint256 p = r - e; // profit

        // Compute revenue share
        uint256 s = Math.max(
            REVENUE_COEFFICIENT_BASIS_POINTS * r / BASIS_POINT_SCALE,
            PROFIT_COEFFICIENT_BASIS_POINTS * p / BASIS_POINT_SCALE
        ); // share
        uint256 remainder = r - s;

        // Send Beneficiary their revenue share on L2
        require(SafeCall.send(BENEFICIARY, gasleft(), s), "RevenueSharer: Failed to send funds to Beneficiary");

        // Send remaining funds to L1 wallet on L1
        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: remainder }(
            L1_WALLET, WITHDRAWAL_MIN_GAS, bytes("")
        );

        emit FeesDisbursed(lastDisbursementTime, s, r);
    }

    /**
     * @dev Returns the RevenueSharer's best estimate of L1 Fee expenditure for the current accounting period.
     * @dev TODO this just returns zero for now, until L1 Fee Expenditure can be tracked on L2.
     */
    function getL1FeeExpenditure() public pure returns (uint256) {
        return 0;
    }

    /**
     * @dev Receives ETH fees withdrawn from L2 FeeVaults.
     * @dev Will revert if ETH is not sent from L2 FeeVaults.
     */
    receive() external payable virtual {
        if (
            msg.sender != Predeploys.SEQUENCER_FEE_WALLET && msg.sender != Predeploys.BASE_FEE_VAULT
                && msg.sender != Predeploys.L1_FEE_VAULT
        ) {
            revert("RevenueSharer: Only FeeVaults can send ETH to FeeDisburser");
        }
        emit FeesReceived(msg.sender, msg.value);
    }

    /*//////////////////////////////////////////////////////////////
                            Internal Functions
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Withdraws fees from a FeeVault and returns the amount withdrawn.
     * @param _feeVault The address of the FeeVault to withdraw from.
     * @dev Withdrawal will only occur if the given FeeVault's balance is greater than or equal to
     *        the minimum withdrawal amount.
     */
    function feeVaultWithdrawal(address payable _feeVault) internal returns (uint256) {
        require(
            FeeVault(_feeVault).WITHDRAWAL_NETWORK() == FeeVault.WithdrawalNetwork.L2,
            "RevenueSharer: FeeVault must withdraw to L2"
        );
        require(
            FeeVault(_feeVault).RECIPIENT() == address(this),
            "RevenueSharer: FeeVault must withdraw to RevenueSharer contract"
        );
        uint256 initial_balance = address(this).balance;
        if (_feeVault.balance >= FeeVault(_feeVault).MIN_WITHDRAWAL_AMOUNT()) {
            FeeVault(_feeVault).withdraw(); // TODO do we need a reentrancy guard around this?
        }
        return address(this).balance - initial_balance;
    }
}
