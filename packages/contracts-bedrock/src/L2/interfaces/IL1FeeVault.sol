// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IFeeVault } from "src/universal/interfaces/IFeeVault.sol";

interface IL1FeeVault {
    event Withdrawal(uint256 value, address to, address from);
    event Withdrawal(uint256 value, address to, address from, IFeeVault.WithdrawalNetwork withdrawalNetwork);

    receive() external payable;

    function MIN_WITHDRAWAL_AMOUNT() external view returns (uint256);
    function RECIPIENT() external view returns (address);
    function WITHDRAWAL_NETWORK() external view returns (IFeeVault.WithdrawalNetwork);
    function minWithdrawalAmount() external view returns (uint256 amount_);
    function recipient() external view returns (address recipient_);
    function totalProcessed() external view returns (uint256);
    function withdraw() external;
    function withdrawalNetwork() external view returns (IFeeVault.WithdrawalNetwork network_);

    function version() external view returns (string memory);

    function __constructor__(
        address _recipient,
        uint256 _minWithdrawalAmount,
        IFeeVault.WithdrawalNetwork _withdrawalNetwork
    )
        external;
}
