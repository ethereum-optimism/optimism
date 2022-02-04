// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

/**
 * @title TeleportrDeposit
 *
 * Shout out to 0xclem for providing the inspiration for this contract:
 * https://github.com/0xclem/teleportr/blob/main/contracts/BridgeDeposit.sol
 */
contract TeleportrDeposit {
    address public owner;
    uint256 public minDepositAmount;
    uint256 public maxDepositAmount;
    uint256 public maxBalance;
    uint256 public totalDeposits;

    // Events
    event OwnerSet(address indexed oldOwner, address indexed newOwner);
    event MinDepositAmountSet(uint256 previousAmount, uint256 newAmount);
    event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount);
    event MaxBalanceSet(uint256 previousBalance, uint256 newBalance);
    event BalanceWithdrawn(address indexed owner, uint256 balance);
    event EtherReceived(uint256 indexed depositId, address indexed emitter, uint256 indexed amount);

    // Modifiers
    modifier isOwner() {
        require(msg.sender == owner, "Caller is not owner");
        _;
    }

    constructor(
        uint256 _minDepositAmount,
        uint256 _maxDepositAmount,
        uint256 _maxBalance
    ) {
        owner = msg.sender;
        minDepositAmount = _minDepositAmount;
        maxDepositAmount = _maxDepositAmount;
        maxBalance = _maxBalance;
        totalDeposits = 0;
        emit OwnerSet(address(0), msg.sender);
        emit MinDepositAmountSet(0, _minDepositAmount);
        emit MaxDepositAmountSet(0, _maxDepositAmount);
        emit MaxBalanceSet(0, _maxBalance);
    }

    // Receive function which reverts if the amount is outside the range
    // [minDepositAmount, maxDepositAmount], or the amount would put the
    // contract over its maxBalance.
    receive() external payable {
        require(msg.value >= minDepositAmount, "Deposit amount is too small");
        require(msg.value <= maxDepositAmount, "Deposit amount is too big");
        require(address(this).balance <= maxBalance, "Contract max balance exceeded");

        emit EtherReceived(totalDeposits, msg.sender, msg.value);
        unchecked {
            totalDeposits += 1;
        }
    }

    // Send the contract's balance to the owner
    function withdrawBalance() external isOwner {
        uint256 _balance = address(this).balance;
        emit BalanceWithdrawn(owner, _balance);
        payable(owner).transfer(_balance);
    }

    // Setters
    function setMinAmount(uint256 _minDepositAmount) external isOwner {
        emit MinDepositAmountSet(minDepositAmount, _minDepositAmount);
        minDepositAmount = _minDepositAmount;
    }

    function setMaxAmount(uint256 _maxDepositAmount) external isOwner {
        emit MaxDepositAmountSet(maxDepositAmount, _maxDepositAmount);
        maxDepositAmount = _maxDepositAmount;
    }

    function setOwner(address newOwner) external isOwner {
        emit OwnerSet(owner, newOwner);
        owner = newOwner;
    }

    function setMaxBalance(uint256 _maxBalance) external isOwner {
        emit MaxBalanceSet(maxBalance, _maxBalance);
        maxBalance = _maxBalance;
    }
}
