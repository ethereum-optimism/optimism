// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Math } from "@openzeppelin/contracts/utils/math/Math.sol";
import { Initializable } from "@openzeppelin/contracts/proxy/utils/Initializable.sol";

import { L2StandardBridge } from "src/L2/L2StandardBridge.sol";
import { FeeVault } from "src/universal/FeeVault.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

error OnlyFeeVaults();
error ZeroAddress(string);
error UnexpectedFeeVaultWithdrawalNetwork();
error UnexpectedFeeVaultRecipient();
error FailedToShare();

/// @custom:proxied
/// @custom:predeploy 0x4200000000000000000000000000000000000024
/// @title RevenueSharer
/// @dev Withdraws funds from system FeeVault contracts,
/// pays a share of revenue to a designated Beneficiary
/// and sends the remainder to a configurable adddress on L1.
contract RevenueSharer is Initializable {
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
     * @dev The percentage coefficient of profit denominated in bass points that is used in
     *      Optimism revenue share calculation.
     */
    uint256 public constant PROFIT_COEFFICIENT_BASIS_POINTS = 250;

    /*//////////////////////////////////////////////////////////////
                            Immutables
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev The address of the Optimism wallet that will receive Optimism's revenue share.
     */
    address payable public BENEFICIARY;
    /**
     * @dev The address of the L1 wallet that will receive the OP chain runner's share of fees.
     */
    address public L1_WALLET;

    /*//////////////////////////////////////////////////////////////
                            Events
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Emitted when fees are disbursed.
     * @param _share The amount of fees shared to the Beneficiary.
     * @param _total The total funds distributed.
     */
    event FeesDisbursed(uint256 _share, uint256 _total);
    /**
     * @dev Emitted when fees are received from FeeVaults.
     * @param _sender The FeeVault that sent the fees.
     * @param _amount The amount of fees received.
     */
    event FeesReceived(address indexed _sender, uint256 _amount);

    /*//////////////////////////////////////////////////////////////
                            Constructor
    //////////////////////////////////////////////////////////////*/
    /**
     * @dev Constructor for the FeeDisburser contract which validates and sets immutable variables.
     * @param _beneficiary The address which receives the revenue share.
     * @param _l1Wallet The L1 address which receives the remainder of the revenue.
     */
    constructor(address payable _beneficiary, address payable _l1Wallet) {
        initialize(_beneficiary, _l1Wallet);
    }

    function initialize(address payable _beneficiary, address payable _l1Wallet) public initializer {
        if (_beneficiary == address(0)) revert ZeroAddress("_beneficiary");
        if (_l1Wallet == address(0)) revert ZeroAddress("_l1Wallet");
        BENEFICIARY = _beneficiary;
        L1_WALLET = _l1Wallet;
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
        if (!SafeCall.send(BENEFICIARY, gasleft(), s)) {
            revert FailedToShare();
        }

        // Send remaining funds to L1 wallet on L1
        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: remainder }(
            L1_WALLET, WITHDRAWAL_MIN_GAS, bytes("")
        );

        emit FeesDisbursed(s, r);
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
            revert OnlyFeeVaults();
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
        if (FeeVault(_feeVault).WITHDRAWAL_NETWORK() != FeeVault.WithdrawalNetwork.L2) {
            revert UnexpectedFeeVaultWithdrawalNetwork();
        }
        if (FeeVault(_feeVault).RECIPIENT() != address(this)) revert UnexpectedFeeVaultRecipient();
        uint256 initial_balance = address(this).balance;
        // The following line will call back into the receive() function on this contract,
        // causing all of the ether from the fee vault to move to this contract:
        FeeVault(_feeVault).withdraw();
        return address(this).balance - initial_balance;
    }
}
