// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface IL1Withdrawer is ISemver {
    error L1Withdrawer_OnlyProxyAdminOwner();

    event WithdrawalInitiated(address indexed recipient, uint256 amount);
    event FundsReceived(address indexed sender, uint256 amount, uint256 newBalance);
    event MinWithdrawalAmountUpdated(uint256 oldMinWithdrawalAmount, uint256 newMinWithdrawalAmount);
    event RecipientUpdated(address oldRecipient, address newRecipient);
    event WithdrawalGasLimitUpdated(uint32 oldWithdrawalGasLimit, uint32 newWithdrawalGasLimit);

    function minWithdrawalAmount() external view returns (uint256);
    function recipient() external view returns (address);
    function withdrawalGasLimit() external view returns (uint32);

    function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) external;
    function setRecipient(address _newRecipient) external;
    function setWithdrawalGasLimit(uint32 _newWithdrawalGasLimit) external;

    receive() external payable;

    function __constructor__(uint256 _minWithdrawalAmount, address _recipient, uint32 _withdrawalGasLimit) external;
}
