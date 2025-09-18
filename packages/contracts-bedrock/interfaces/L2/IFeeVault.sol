// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { Types } from "src/libraries/Types.sol";

interface IFeeVault {
    error FeeVault_OnlyProxyAdminOwner();

    event Withdrawal(uint256 value, address to, address from);
    event Withdrawal(uint256 value, address to, address from, Types.WithdrawalNetwork withdrawalNetwork);
    event MinWithdrawalAmountUpdated(uint256 oldWithdrawalAmount, uint256 newWithdrawalAmount);
    event RecipientUpdated(address oldRecipient, address newRecipient);
    event WithdrawalNetworkUpdated(
        Types.WithdrawalNetwork oldWithdrawalNetwork, Types.WithdrawalNetwork newWithdrawalNetwork
    );

    receive() external payable;

    function MIN_WITHDRAWAL_AMOUNT() external view returns (uint256);
    function RECIPIENT() external view returns (address);
    function WITHDRAWAL_NETWORK() external view returns (Types.WithdrawalNetwork);
    function minWithdrawalAmount() external view returns (uint256);
    function recipient() external view returns (address);
    function totalProcessed() external view returns (uint256);
    function withdraw() external returns (uint256 value_);
    function withdrawalNetwork() external view returns (Types.WithdrawalNetwork);
    function setMinWithdrawalAmount(uint256 _newMinWithdrawalAmount) external;
    function setRecipient(address _newRecipient) external;
    function setWithdrawalNetwork(Types.WithdrawalNetwork _newWithdrawalNetwork) external;

    function __constructor__() external;
}

interface IFeeVaultConstructor {
    /// NOTE: This is the real constructor for the FeeVault contract, but can't be added to the main interface because
    ///       it is an abstract contract, and on the scripts it gets an empty constructor automatically generated that
    ///       makes the `interfaces-check` script fail.
    function __constructor__(
        address _recipient,
        uint256 _minWithdrawalAmount,
        Types.WithdrawalNetwork _withdrawalNetwork
    )
        external;
}
