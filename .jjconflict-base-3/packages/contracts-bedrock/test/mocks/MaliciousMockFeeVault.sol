// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Types } from "src/libraries/Types.sol";

/// @notice Mock fee vault that returns a different withdrawal amount than what it transfers
contract MaliciousMockFeeVault {
    address public immutable RECIPIENT;
    uint256 public immutable ACTUAL_TRANSFER_AMOUNT;
    uint256 public immutable CLAIMED_WITHDRAWAL_AMOUNT;

    constructor(address payable _recipient, uint256 _actualTransferAmount, uint256 _claimedWithdrawalAmount) {
        RECIPIENT = _recipient;
        ACTUAL_TRANSFER_AMOUNT = _actualTransferAmount;
        CLAIMED_WITHDRAWAL_AMOUNT = _claimedWithdrawalAmount;
    }

    receive() external payable { }

    function withdrawalNetwork() external pure returns (Types.WithdrawalNetwork) {
        return Types.WithdrawalNetwork.L2;
    }

    function recipient() external view returns (address) {
        return RECIPIENT;
    }

    function withdraw() external returns (uint256) {
        // Transfer the actual amount
        (bool success,) = RECIPIENT.call{ value: ACTUAL_TRANSFER_AMOUNT }("");
        require(success, "MaliciousMockFeeVault: failed to send ETH");

        // But lie about how much was transferred
        return CLAIMED_WITHDRAWAL_AMOUNT;
    }
}
