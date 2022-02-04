// SPDX-License-Identifier: MIT
pragma solidity >=0.8.9;

/**
 * @title TeleportrDeposit
 *
 * Shout out to 0xclem for providing the inspiration for this contract:
 * https://github.com/0xclem/teleportr/blob/main/contracts/BridgeDeposit.sol
 */
contract TeleportrDeposit {
    address private owner;
    uint256 private maxDepositAmount;
    uint256 private maxBalance;
    bool private canReceiveDeposit;

    // Events
    event OwnerSet(address indexed oldOwner, address indexed newOwner);
    event MaxDepositAmountSet(uint256 previousAmount, uint256 newAmount);
    event CanReceiveDepositSet(bool canReceiveDeposit);
    event MaxBalanceSet(uint256 previousBalance, uint256 newBalance);
    event BalanceWithdrawn(address indexed owner, uint256 balance);
    event EtherReceived(address indexed emitter, uint256 amount);
    event Destructed(address indexed owner, uint256 amount);

    // Modifiers
    modifier isLowerThanMaxDepositAmount() {
        require(msg.value <= maxDepositAmount, "Deposit amount is too big");
        _;
    }
    modifier isOwner() {
        require(msg.sender == owner, "Caller is not owner");
        _;
    }
    modifier canReceive() {
        require(canReceiveDeposit == true, "Contract is not allowed to receive ether");
        _;
    }
    modifier isLowerThanMaxBalance() {
        require(address(this).balance <= maxBalance, "Contract reached the max balance allowed");
        _;
    }

    constructor(
        uint256 _maxDepositAmount,
        uint256 _maxBalance,
        bool _canReceiveDeposit
    ) {
        owner = msg.sender;
        maxDepositAmount = _maxDepositAmount;
        maxBalance = _maxBalance;
        canReceiveDeposit = _canReceiveDeposit;
        emit OwnerSet(address(0), msg.sender);
        emit MaxDepositAmountSet(0, _maxDepositAmount);
        emit MaxBalanceSet(0, _maxBalance);
        emit CanReceiveDepositSet(_canReceiveDeposit);
    }

    // Receive function which reverts if amount > maxDepositAmount and canReceiveDeposit = false
    receive() external payable isLowerThanMaxDepositAmount canReceive isLowerThanMaxBalance {
        emit EtherReceived(msg.sender, msg.value);
    }

    // Send the contract's balance to the owner
    function withdrawBalance() external isOwner {
        uint256 _balance = address(this).balance;
        emit BalanceWithdrawn(owner, _balance);
        payable(owner).transfer(_balance);
    }

    function destroy() public isOwner {
        emit Destructed(owner, address(this).balance);
        selfdestruct(payable(owner));
    }

    // Setters
    function setMaxAmount(uint256 _maxDepositAmount) external isOwner {
        emit MaxDepositAmountSet(maxDepositAmount, _maxDepositAmount);
        maxDepositAmount = _maxDepositAmount;
    }

    function setOwner(address newOwner) external isOwner {
        emit OwnerSet(owner, newOwner);
        owner = newOwner;
    }

    function setCanReceiveDeposit(bool _canReceiveDeposit) external isOwner {
        emit CanReceiveDepositSet(_canReceiveDeposit);
        canReceiveDeposit = _canReceiveDeposit;
    }

    function setMaxBalance(uint256 _maxBalance) external isOwner {
        emit MaxBalanceSet(maxBalance, _maxBalance);
        maxBalance = _maxBalance;
    }

    // Getters
    function getMaxDepositAmount() external view returns (uint256) {
        return maxDepositAmount;
    }

    function getMaxBalance() external view returns (uint256) {
        return maxBalance;
    }

    function getOwner() external view returns (address) {
        return owner;
    }

    function getCanReceiveDeposit() external view returns (bool) {
        return canReceiveDeposit;
    }
}
