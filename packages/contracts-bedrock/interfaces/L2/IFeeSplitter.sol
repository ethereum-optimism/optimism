// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";
import { ISharesCalculator } from "interfaces/L2/ISharesCalculator.sol";

interface IFeeSplitter is ISemver {
    error FeeSplitter_ExceedsMaxFeeDisbursementTime();
    error FeeSplitter_SharesCalculatorCannotBeZero();
    error FeeSplitter_DisbursementIntervalNotReached();
    error FeeSplitter_FeeShareInfoEmpty();
    error FeeSplitter_NoFeesCollected();
    error FeeSplitter_FeeVaultMustWithdrawToL2();
    error FeeSplitter_FeeVaultMustWithdrawToFeeSplitter();
    error FeeSplitter_OnlyProxyAdminOwner();
    error FeeSplitter_FailedToSendToRevenueShareRecipient();
    error FeeSplitter_SharesCalculatorMalformedOutput();
    error FeeSplitter_ReceiveWindowClosed();
    error FeeSplitter_SenderNotApprovedVault();

    event FeesReceived(address indexed sender, uint256 amount);
    event FeeDisbursementIntervalUpdated(uint128 oldFeeDisbursementInterval, uint128 newFeeDisbursementInterval);
    event FeesDisbursed(ISharesCalculator.ShareInfo[] shareInfo, uint256 grossRevenue);
    event SharesCalculatorUpdated(address oldSharesCalculator, address newSharesCalculator);

    function sharesCalculator() external view returns (ISharesCalculator);
    function lastDisbursementTime() external view returns (uint128);
    function feeDisbursementInterval() external view returns (uint128);

    function initialize(ISharesCalculator _sharesCalculator) external;

    function disburseFees() external;

    function setFeeDisbursementInterval(uint128 _newFeeDisbursementInterval) external;

    function setSharesCalculator(ISharesCalculator _newSharesCalculator) external;

    receive() external payable;
}
