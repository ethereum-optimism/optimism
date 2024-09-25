// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

interface IFeeVault {
    enum WithdrawalNetwork {
        L1,
        L2
    }

    event Withdrawal(uint256 value, address to, address from);
    event Withdrawal(uint256 value, address to, address from, WithdrawalNetwork withdrawalNetwork);

    receive() external payable;

    function MIN_WITHDRAWAL_AMOUNT() external view returns (uint256);
    function RECIPIENT() external view returns (address);
    function WITHDRAWAL_NETWORK() external view returns (WithdrawalNetwork);
    function minWithdrawalAmount() external view returns (uint256 amount_);
    function recipient() external view returns (address recipient_);
    function totalProcessed() external view returns (uint256);
    function withdraw() external;
    function withdrawalNetwork() external view returns (WithdrawalNetwork network_);
}
