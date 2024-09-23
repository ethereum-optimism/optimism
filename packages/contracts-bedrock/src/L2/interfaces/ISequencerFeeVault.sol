// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IFeeVault } from "src/universal/interfaces/IFeeVault.sol";

interface ISequencerFeeVault is IFeeVault {
    function version() external view returns (string memory);
    function l1FeeWallet() external view returns (address);

    function __constructor__(
        address _recipient,
        uint256 _minWithdrawalAmount,
        WithdrawalNetwork _withdrawalNetwork
    )
        external;
}
