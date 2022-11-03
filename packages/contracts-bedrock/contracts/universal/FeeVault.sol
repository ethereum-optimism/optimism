// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

/**
 * @title FeeVault
 * @notice The FeeVault contract has the base logic for handling transaction fees.
 */
abstract contract FeeVault {
    /**
     * @notice Emits each time that a withdrawal occurs
     */
    event Withdrawal(uint256 value, address to, address from);

    /**
     * @notice Minimum balance before a withdrawal can be triggered.
     */
    uint256 public immutable MIN_WITHDRAWAL_AMOUNT;

    /**
     * @notice Wallet that will receive the fees on L1.
     */
    address public immutable RECIPIENT;

    /**
     * @param _recipient - The L1 account that funds can be withdrawn to.
     * @param _minWithdrawalAmount - The min amount of funds before a withdrawal
     *        can be triggered.
     */
    constructor(address _recipient, uint256 _minWithdrawalAmount) {
        MIN_WITHDRAWAL_AMOUNT = _minWithdrawalAmount;
        RECIPIENT = _recipient;
    }

    /**
     * @notice Allow the contract to receive ETH.
     */
    receive() external payable {}

    /**
     * @notice Triggers a withdrawal of funds to the L1 fee wallet.
     */
    function withdraw() external {
        require(
            address(this).balance >= MIN_WITHDRAWAL_AMOUNT,
            "FeeVault: withdrawal amount must be greater than minimum withdrawal amount"
        );

        uint256 value = address(this).balance;
        emit Withdrawal(value, RECIPIENT, msg.sender);

        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: value }(
            RECIPIENT,
            20000,
            bytes("")
        );
    }
}
