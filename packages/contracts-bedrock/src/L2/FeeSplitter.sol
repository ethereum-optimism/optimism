// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";
import { Types } from "src/libraries/Types.sol";
import { SafeCall } from "src/libraries/SafeCall.sol";

// Interfaces
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";
import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";
import { IFeeVault } from "interfaces/L2/IFeeVault.sol";

// OpenZeppelin
import { Initializable } from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

/// @custom:proxied
/// @custom:predeploy 0x420000000000000000000000000000000000002B
/// @title FeeSplitter
/// @notice Withdraws funds from system FeeVault contracts and distributes them according to the
///         configured SharesCalculator.
contract FeeSplitter is ISemver, Initializable {
    /// @notice Thrown when the fee disbursement interval exceeds the maximum allowed.
    error FeeSplitter_ExceedsMaxFeeDisbursementTime();

    /// @notice Thrown when the fee disbursement interval is set to zero.
    error FeeSplitter_FeeDisbursementIntervalCannotBeZero();

    /// @notice Thrown when the share calculator address is zero.
    error FeeSplitter_SharesCalculatorCannotBeZero();

    /// @notice Thrown when the disbursement interval has not been reached.
    error FeeSplitter_DisbursementIntervalNotReached();

    /// @notice Thrown when the fee share recipients are empty.
    error FeeSplitter_FeeShareInfoEmpty();

    /// @notice Thrown when no fees are collected from vaults during disbursement.
    error FeeSplitter_NoFeesCollected();

    /// @notice Thrown when the FeeVault does not withdraw to L2.
    error FeeSplitter_FeeVaultMustWithdrawToL2();

    /// @notice Thrown when the FeeVault does not withdraw to FeeSplitter contract.
    error FeeSplitter_FeeVaultMustWithdrawToFeeSplitter();

    /// @notice Thrown when the FeeVault withdrawal amount does not match the expected amount.
    error FeeSplitter_FeeVaultWithdrawalAmountMismatch();

    /// @notice Thrown when the caller is not the ProxyAdmin owner.
    error FeeSplitter_OnlyProxyAdminOwner();

    /// @notice Thrown when sending funds to the fee recipient fails.
    error FeeSplitter_FailedToSendToRevenueShareRecipient();

    /// @notice Thrown when the sharesCalculator returns malformed output.
    error FeeSplitter_SharesCalculatorMalformedOutput();

    /// @notice Thrown when the sender is not the currently disbursing vault.
    error FeeSplitter_SenderNotCurrentVault();

    /// @notice Transient storage slot key for the address of the vault currently allowed to disburse.
    ///         Equal to bytes32(uint256(keccak256("feesplitter.disbursingAddress")) - 1)
    bytes32 internal constant _FEE_SPLITTER_DISBURSING_ADDRESS_SLOT =
        0x21346dddac42cc163a6523eefc19df981df7352c870dc3b0b17a6a92fc6fe813;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0
    string public constant version = "1.0.0";

    /// @notice max time between fee disbursements
    uint128 public constant MAX_DISBURSEMENT_INTERVAL = 365 days;

    /// @notice The contract which determines the recipients and their weights for fee disbursement.
    ISharesCalculator public sharesCalculator;

    /// @notice The timestamp of the last disbursal.
    uint128 public lastDisbursementTime;

    /// @notice The minimum amount of time in seconds that must pass between fee disbursal.
    uint128 public feeDisbursementInterval;

    /// @notice Emitted when fees are received from FeeVaults.
    /// @param sender The FeeVault that sent the fees.
    /// @param amount The amount of fees received.
    /// @param newBalance The new balance after receiving fees.
    event FeesReceived(address indexed sender, uint256 amount, uint256 newBalance);

    /// @notice Emitted when the fee disbursement interval is updated.
    /// @param oldFeeDisbursementInterval The previous fee disbursement interval.
    /// @param newFeeDisbursementInterval The new fee disbursement interval.
    event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval);

    /// @notice Emitted when fees are disbursed to the recipients.
    /// @param shareInfo The recipients of the fee share.
    /// @param grossRevenue The gross revenue before disbursement.
    event FeesDisbursed(ISharesCalculator.ShareInfo[] shareInfo, uint256 grossRevenue);

    /// @notice Emitted when the share calculator is updated.
    /// @param oldSharesCalculator The old share calculator contract.
    /// @param newSharesCalculator The new share calculator contract.
    event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator);

    constructor() {
        _disableInitializers();
    }

    /// @notice Initializes the contract with all required addresses and parameters.
    /// @dev This function can only be called once and must be called by the ProxyAdmin owner.
    /// @param _sharesCalculator            The share calculator contract.
    function initialize(ISharesCalculator _sharesCalculator) external initializer {
        sharesCalculator = _sharesCalculator;
        // As default, the fee disbursement interval is 1 day
        feeDisbursementInterval = 1 days;

        // Set the last disbursement time to the current block timestamp
        lastDisbursementTime = uint128(block.timestamp);
    }

    /// @dev Receives ETH fees withdrawn from L2 FeeVaults.
    receive() external payable virtual {
        // Sender must be the currently disbursing vault
        if (msg.sender != _getTransientDisbursingAddress()) {
            revert FeeSplitter_SenderNotCurrentVault();
        }

        uint256 newBalance = address(this).balance;
        emit FeesReceived(msg.sender, msg.value, newBalance);
    }

    /// @notice Withdraws funds from FeeVaults and disburses them to the recipients.
    function disburseFees() external {
        if (block.timestamp < lastDisbursementTime + feeDisbursementInterval) {
            revert FeeSplitter_DisbursementIntervalNotReached();
        }

        // Update the last disbursement time
        lastDisbursementTime = uint128(block.timestamp);

        // Pull fees into the contract
        uint256 sequencerFees = _feeVaultWithdrawal(payable(Predeploys.SEQUENCER_FEE_WALLET));
        uint256 baseFees = _feeVaultWithdrawal(payable(Predeploys.BASE_FEE_VAULT));
        uint256 l1Fees = _feeVaultWithdrawal(payable(Predeploys.L1_FEE_VAULT));
        uint256 operatorFees = _feeVaultWithdrawal(payable(Predeploys.OPERATOR_FEE_VAULT));
        // Clear the transient disbursing address
        _setTransientDisbursingAddress(address(0));

        uint256 grossRevenue = sequencerFees + baseFees + operatorFees + l1Fees;

        // Revert if no fees were collected
        if (grossRevenue == 0) {
            revert FeeSplitter_NoFeesCollected();
        }

        // Call to the sharesCalculator to determine the fee share recipients, amounts, withdrawal networks, and data
        // DoS risk if array size is too large.
        (ISharesCalculator.ShareInfo[] memory shareInfo) =
            sharesCalculator.getRecipientsAndAmounts(sequencerFees, baseFees, operatorFees, l1Fees);

        uint256 shareInfoLength = shareInfo.length;

        // Ensure the share calculator returned valid data
        if (shareInfoLength == 0) revert FeeSplitter_FeeShareInfoEmpty();

        // Loop through the recipients and their corresponding fee shares
        uint256 totalFeesDisbursed;
        for (uint256 i; i < shareInfoLength; i++) {
            uint256 feesAmount = shareInfo[i].amount;

            // Ensure the fee share is greater than zero
            if (feesAmount == 0) continue;

            bool success = SafeCall.send(shareInfo[i].recipient, feesAmount);
            if (!success) {
                revert FeeSplitter_FailedToSendToRevenueShareRecipient();
            }
            totalFeesDisbursed += feesAmount;
        }

        // Ensure the total fees disbursed is equal to the gross revenue
        /// NOTE: Contract can hold some balance after disbursement if tokens are force sent (using SELFDESTRUCT).
        if (totalFeesDisbursed != grossRevenue) revert FeeSplitter_SharesCalculatorMalformedOutput();

        emit FeesDisbursed({ shareInfo: shareInfo, grossRevenue: grossRevenue });
    }

    /// @notice Updates the fee disbursement interval. Only callable by the ProxyAdmin owner.
    /// @param _newFeeDisbursementInterval The new fee disbursement interval in seconds.
    function setFeeDisbursementInterval(uint128 _newFeeDisbursementInterval) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert FeeSplitter_OnlyProxyAdminOwner();
        }
        if (_newFeeDisbursementInterval == 0) {
            revert FeeSplitter_FeeDisbursementIntervalCannotBeZero();
        }
        if (_newFeeDisbursementInterval > MAX_DISBURSEMENT_INTERVAL) {
            revert FeeSplitter_ExceedsMaxFeeDisbursementTime();
        }
        uint128 oldFeeDisbursementInterval = feeDisbursementInterval;
        feeDisbursementInterval = _newFeeDisbursementInterval;
        emit FeeDisbursementIntervalUpdated(oldFeeDisbursementInterval, _newFeeDisbursementInterval);
    }

    /// @notice Updates the share calculator contract. Only callable by the ProxyAdmin owner.
    /// @param _newSharesCalculator The new share calculator contract.
    function setSharesCalculator(ISharesCalculator _newSharesCalculator) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) {
            revert FeeSplitter_OnlyProxyAdminOwner();
        }
        if (address(_newSharesCalculator) == address(0)) revert FeeSplitter_SharesCalculatorCannotBeZero();
        address oldSharesCalculator = address(sharesCalculator);
        sharesCalculator = _newSharesCalculator;
        emit SharesCalculatorUpdated(oldSharesCalculator, address(_newSharesCalculator));
    }

    /// @notice Checks & Withdraws fees from a FeeVault.
    /// @dev Withdrawal will only occur if the vault is properly configured.
    ///      The FeeVault itself will enforce minimum withdrawal requirements.
    /// @param _feeVault The address of the FeeVault to withdraw from.
    /// @return value_ The amount of ETH that was withdrawn from the vault.
    function _feeVaultWithdrawal(address payable _feeVault) internal returns (uint256 value_) {
        if (IFeeVault(_feeVault).withdrawalNetwork() != Types.WithdrawalNetwork.L2) {
            revert FeeSplitter_FeeVaultMustWithdrawToL2();
        }
        if (IFeeVault(_feeVault).recipient() != address(this)) {
            revert FeeSplitter_FeeVaultMustWithdrawToFeeSplitter();
        }

        uint256 balanceBefore = address(this).balance;
        _setTransientDisbursingAddress(address(_feeVault));
        value_ = IFeeVault(_feeVault).withdraw();
        uint256 balanceAfter = address(this).balance;

        if (balanceAfter - balanceBefore != value_) {
            revert FeeSplitter_FeeVaultWithdrawalAmountMismatch();
        }
    }

    /// @notice Sets the transient disbursing address.
    /// @param _allowedCaller The address of the vault allowed to call receive().
    function _setTransientDisbursingAddress(address _allowedCaller) internal {
        assembly {
            tstore(_FEE_SPLITTER_DISBURSING_ADDRESS_SLOT, _allowedCaller)
        }
    }

    /// @notice Reads the transient disbursing address.
    /// @return allowedCaller_ The address of the vault currently allowed to call receive().
    function _getTransientDisbursingAddress() internal view returns (address allowedCaller_) {
        assembly {
            allowedCaller_ := tload(_FEE_SPLITTER_DISBURSING_ADDRESS_SLOT)
        }
    }
}
