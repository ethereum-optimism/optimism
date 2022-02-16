// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @title TeleportrDeposit
 *
 * Shout out to 0xclem for providing the inspiration for this contract:
 * https://github.com/0xclem/teleportr/blob/main/contracts/BridgeDeposit.sol
 */
contract TeleportrDeposit is Ownable {
    /// The minimum amount that be deposited in a receive.
    uint256 public minDepositAmount;
    /// The maximum amount that be deposited in a receive.
    uint256 public maxDepositAmount;
    /// The maximum balance the contract can hold after a receive.
    uint256 public maxBalance;
    /// The total number of successful deposits received.
    uint256 public totalDeposits;

    /**
     * @notice Emitted any time the minimum deposit amount is set.
     * @param previousAmount The previous minimum deposit amount.
     * @param newAmount The new minimum deposit amount.
     */
    event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount);

    /**
     * @notice Emitted any time the maximum deposit amount is set.
     * @param previousAmount The previous maximum deposit amount.
     * @param newAmount The new maximum deposit amount.
     */
    event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount);

    /**
     * @notice Emitted any time the contract maximum balance is set.
     * @param previousBalance The previous maximum contract balance.
     * @param newBalance The new maximum contract balance.
     */
    event MaxBalanceSet(uint256 previousBalance, uint256 newBalance);

    /**
     * @notice Emitted any time the balance is withdrawn by the owner.
     * @param owner The current owner and recipient of the funds.
     * @param balance The current contract balance paid to the owner.
     */
    event BalanceWithdrawn(address indexed owner, uint256 balance);

    /**
     * @notice Emitted any time a successful deposit is received.
     * @param depositId A unique sequencer number identifying the deposit.
     * @param emitter The sending address of the payer.
     * @param amount The amount deposited by the payer.
     */
    event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount);

    /**
     * @notice Initializes a new TeleportrDeposit contract.
     * @param _minDepositAmount The initial minimum deposit amount.
     * @param _maxDepositAmount The initial maximum deposit amount.
     * @param _maxBalance The initial maximum contract balance.
     */
    constructor(
        uint256 _minDepositAmount,
        uint256 _maxDepositAmount,
        uint256 _maxBalance
    ) {
        minDepositAmount = _minDepositAmount;
        maxDepositAmount = _maxDepositAmount;
        maxBalance = _maxBalance;
        totalDeposits = 0;
        emit MinDepositAmountSet(0, _minDepositAmount);
        emit MaxDepositAmountSet(0, _maxDepositAmount);
        emit MaxBalanceSet(0, _maxBalance);
    }

    /**
     * @notice Accepts deposits that will be disbursed to the sender's address on L2.
     * The method reverts if the amount is less than the current
     * minDepositAmount, the amount is greater than the current
     * maxDepositAmount, or the amount causes the contract to exceed its maximum
     * allowed balance.
     */
    receive() external payable {
        require(msg.value >= minDepositAmount, "Deposit amount is too small");
        require(msg.value <= maxDepositAmount, "Deposit amount is too big");
        require(address(this).balance <= maxBalance, "Contract max balance exceeded");

        emit EtherReceived(totalDeposits, msg.sender, msg.value);
        unchecked {
            totalDeposits += 1;
        }
    }

    /**
     * @notice Sends the contract's current balance to the owner.
     */
    function withdrawBalance() external onlyOwner {
        address _owner = owner();
        uint256 _balance = address(this).balance;
        emit BalanceWithdrawn(_owner, _balance);
        payable(_owner).transfer(_balance);
    }

    /**
     * @notice Sets the minimum amount that can be deposited in a receive.
     * @param _minDepositAmount The new minimum deposit amount.
     */
    function setMinAmount(uint256 _minDepositAmount) external onlyOwner {
        emit MinDepositAmountSet(minDepositAmount, _minDepositAmount);
        minDepositAmount = _minDepositAmount;
    }

    /**
     * @notice Sets the maximum amount that can be deposited in a receive.
     * @param _maxDepositAmount The new maximum deposit amount.
     */
    function setMaxAmount(uint256 _maxDepositAmount) external onlyOwner {
        emit MaxDepositAmountSet(maxDepositAmount, _maxDepositAmount);
        maxDepositAmount = _maxDepositAmount;
    }

    /**
     * @notice Sets the maximum balance the contract can hold after a receive.
     * @param _maxBalance The new maximum contract balance.
     */
    function setMaxBalance(uint256 _maxBalance) external onlyOwner {
        emit MaxBalanceSet(maxBalance, _maxBalance);
        maxBalance = _maxBalance;
    }
}
