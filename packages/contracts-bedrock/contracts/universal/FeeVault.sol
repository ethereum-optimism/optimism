// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

import { Semver } from "../universal/Semver.sol";
import { L2StandardBridge } from "../L2/L2StandardBridge.sol";
import { Predeploys } from "../libraries/Predeploys.sol";

/**
 * @title FeeVault
 * @notice The FeeVault contract has the base logic for handling transaction fees.
 */
contract FeeVault {
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
    address public immutable recipient;

    /**
     * @custom:semver 0.0.1
     */
    constructor(address _recipient, uint256 _minWithdrawalAmount) {
        MIN_WITHDRAWAL_AMOUNT = _minWithdrawalAmount;
        recipient = _recipient;
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

        L2StandardBridge(payable(Predeploys.L2_STANDARD_BRIDGE)).bridgeETHTo{ value: value }(
            recipient,
            20000,
            bytes("")
        );

        emit Withdrawal(value, recipient, msg.sender);
    }
}
