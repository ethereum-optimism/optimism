// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "interfaces/universal/ISemver.sol";

interface IL1Withdrawer is ISemver {
    error L1Withdrawer_OnlyProxyAdminOwner();

    event WithdrawalInitiated(uint256 amount, address indexed recipient);
    event MinWithdrawalAmountUpdated(uint256 oldMinWithdrawalAmount, uint256 newMinWithdrawalAmount);
    event RecipientUpdated(address oldRecipient, address newRecipient);
    event WithdrawalGasLimitUpdated(uint96 oldWithdrawalGasLimit, uint96 newWithdrawalGasLimit);

    function minWithdrawalAmount() external view returns (uint256);
    function recipient() external view returns (address);
    function withdrawalGasLimit() external view returns (uint96);

    function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) external;
    function setRecipient(address _newRecipient) external;
    function setWithdrawalGasLimit(uint96 _newWithdrawalGasLimit) external;

    function __constructor__(uint256 _minWithdrawalAmount, address _recipient, uint96 _withdrawalGasLimit) external;
}
