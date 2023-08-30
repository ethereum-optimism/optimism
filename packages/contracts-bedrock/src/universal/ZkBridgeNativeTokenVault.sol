// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import {Initializable} from "@openzeppelin/contracts-upgradeable/proxy/utils/Initializable.sol";

contract ZkBridgeNativeTokenVault is Initializable {
    /// @notice The address authorized for governance interactions.
    address public governor;

    /// @notice The address of the vault manager.
    mapping(address => bool) public managements;

    error Unauthorized();
    error CannotBeZeroAddress();

    event Deposit(address indexed dst, uint256 wad);
    event Withdrawal(address indexed src, uint256 wad);
    event NewGovernor(address indexed governor);
    event NewManager(address indexed manager, bool actived);

    modifier onlyGovernor() {
        if (msg.sender != governor) revert Unauthorized();
        _;
    }

    modifier onlyManagement() {
        if (managements[msg.sender] == true) {
            revert Unauthorized();
        }
        _;
    }

    modifier checkZeroAddress(address address_) {
        if (address_ == address(0)) revert CannotBeZeroAddress();
        _;
    }

    constructor() {}

    function initialize(address governor_) public initializer {
        _setGovernor(governor_);
    }

    receive() external payable {
        deposit();
    }

    function deposit() public payable {
        emit Deposit(msg.sender, msg.value);
    }

    function withdraw(uint256 wad) external onlyManagement {
        payable(msg.sender).transfer(wad);
        emit Withdrawal(msg.sender, wad);
    }

    function totalSupply() public view returns (uint256) {
        return address(this).balance;
    }

    /**
     * @dev Set new manager
     * Requirements:
     *
     * - `manager_` cannot be the zero address.
     * - This may only be called by governance or the guardian.
     *
     * @param manager_  The Manager Address Allowed for Withdrawals.
     * @param actived_ The manager activation status.
     */
    function setManager(address manager_, bool actived_) external onlyGovernor checkZeroAddress(manager_) {
        _setManager(manager_, actived_);
    }

    function _setManager(address manager_, bool actived_) private {
        managements[manager_] = actived_;
        emit NewManager(governor, actived_);
    }

    /**
     * @dev Set new governor
     * Requirements:
     *
     * - `governor_` cannot be the zero address.
     * - This may only be called by governance or the guardian.
     *
     * @param governor_  The governor address which controls the contract
     */
    function setGovernor(address governor_) external onlyGovernor checkZeroAddress(governor_) {
        _setGovernor(governor_);
    }

    function _setGovernor(address governor_) private {
        governor = governor_;
        emit NewGovernor(governor);
    }
}
