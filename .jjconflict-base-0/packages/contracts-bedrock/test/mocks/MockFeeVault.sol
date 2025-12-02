// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Types } from "src/libraries/Types.sol";

/// @notice Simple mock FeeVault for testing that actually transfers ETH
contract MockFeeVault {
    uint256 public immutable MIN_WITHDRAWAL_AMOUNT;
    address public immutable RECIPIENT;
    Types.WithdrawalNetwork public immutable WITHDRAWAL_NETWORK;

    event Withdrawal(uint256 value, address to, address from);
    event Withdrawal(uint256 value, address to, address from, Types.WithdrawalNetwork withdrawalNetwork);

    constructor(address payable _recipient, uint256 _minWithdrawalAmount, Types.WithdrawalNetwork _withdrawalNetwork) {
        RECIPIENT = _recipient;
        MIN_WITHDRAWAL_AMOUNT = _minWithdrawalAmount;
        WITHDRAWAL_NETWORK = _withdrawalNetwork;
    }

    receive() external payable { }

    function withdrawalNetwork() external view returns (Types.WithdrawalNetwork) {
        return WITHDRAWAL_NETWORK;
    }

    function minWithdrawalAmount() external view returns (uint256) {
        return MIN_WITHDRAWAL_AMOUNT;
    }

    function recipient() external view returns (address) {
        return RECIPIENT;
    }

    function withdraw() external returns (uint256) {
        require(
            address(this).balance >= MIN_WITHDRAWAL_AMOUNT,
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        uint256 value = address(this).balance;

        emit Withdrawal(value, RECIPIENT, msg.sender);
        emit Withdrawal(value, RECIPIENT, msg.sender, WITHDRAWAL_NETWORK);

        if (WITHDRAWAL_NETWORK == Types.WithdrawalNetwork.L2) {
            (bool success,) = RECIPIENT.call{ value: value }("");
            require(success, "FeeVault: failed to send ETH to L2 fee recipient");
        }

        return value;
    }
}

/// @title MockLegacyFeeVault
/// @notice Mock fee vault contract that simulates legacy vaults without WITHDRAWAL_NETWORK function
contract MockLegacyFeeVault {
    address public constant RECIPIENT = address(0x1234567890123456789012345678901234567890);
    uint256 public constant MIN_WITHDRAWAL_AMOUNT = 0.01 ether;

    // No WITHDRAWAL_NETWORK() implementation

    receive() external payable { }
}
