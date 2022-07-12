// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

import { Ownable } from "@openzeppelin/contracts/access/Ownable.sol";

/**
 * @custom:attribution https://github.com/0xclem/teleportr
 * @title TeleportrDeposit
 * @notice A contract meant to manage deposits into Optimism's Teleportr custodial bridge. Deposits
 *         are rate limited to avoid a situation where too much ETH is flowing through this bridge
 *         and cannot be properly disbursed on L2. Inspired by 0xclem's original Teleportr system
 *         (https://github.com/0xclem/teleportr).
 */
contract TeleportrDeposit is Ownable {
    /**
     * @notice Minimum deposit amount (in wei).
     */
    uint256 public minDepositAmount;

    /**
     * @notice Maximum deposit amount (in wei).
     */
    uint256 public maxDepositAmount;

    /**
     * @notice Maximum balance this contract will hold before it starts rejecting deposits.
     */
    uint256 public maxBalance;

    /**
     * @notice Total number of deposits received.
     */
    uint256 public totalDeposits;

    /**
     * @notice Emitted any time the minimum deposit amount is set.
     *
     * @param previousAmount The previous minimum deposit amount.
     * @param newAmount      The new minimum deposit amount.
     */
    event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount);

    /**
     * @notice Emitted any time the maximum deposit amount is set.
     *
     * @param previousAmount The previous maximum deposit amount.
     * @param newAmount      The new maximum deposit amount.
     */
    event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount);

    /**
     * @notice Emitted any time the contract maximum balance is set.
     *
     * @param previousBalance The previous maximum contract balance.
     * @param newBalance      The new maximum contract balance.
     */
    event MaxBalanceSet(uint256 previousBalance, uint256 newBalance);

    /**
     * @notice Emitted any time the balance is withdrawn by the owner.
     *
     * @param owner   The current owner and recipient of the funds.
     * @param balance The current contract balance paid to the owner.
     */
    event BalanceWithdrawn(address indexed owner, uint256 balance);

    /**
     * @notice Emitted any time a successful deposit is received.
     *
     * @param depositId A unique sequencer number identifying the deposit.
     * @param emitter   The sending address of the payer.
     * @param amount    The amount deposited by the payer.
     */
    event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount);

    /**
     * @custom:semver 0.0.1
     *
     * @param _minDepositAmount The initial minimum deposit amount.
     * @param _maxDepositAmount The initial maximum deposit amount.
     * @param _maxBalance       The initial maximum contract balance.
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
     *
     * @param _minDepositAmount The new minimum deposit amount.
     */
    function setMinAmount(uint256 _minDepositAmount) external onlyOwner {
        emit MinDepositAmountSet(minDepositAmount, _minDepositAmount);
        minDepositAmount = _minDepositAmount;
    }

    /**
     * @notice Sets the maximum amount that can be deposited in a receive.
     *
     * @param _maxDepositAmount The new maximum deposit amount.
     */
    function setMaxAmount(uint256 _maxDepositAmount) external onlyOwner {
        emit MaxDepositAmountSet(maxDepositAmount, _maxDepositAmount);
        maxDepositAmount = _maxDepositAmount;
    }

    /**
     * @notice Sets the maximum balance the contract can hold after a receive.
     *
     * @param _maxBalance The new maximum contract balance.
     */
    function setMaxBalance(uint256 _maxBalance) external onlyOwner {
        emit MaxBalanceSet(maxBalance, _maxBalance);
        maxBalance = _maxBalance;
    }
}
