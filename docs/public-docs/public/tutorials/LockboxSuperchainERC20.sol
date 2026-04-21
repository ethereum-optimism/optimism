// SPDX-License-Identifier: MIT
pragma solidity ^0.8.25;

import {SuperchainERC20} from "./SuperchainERC20.sol";
import {IERC20} from "@openzeppelin/contracts/interfaces/IERC20.sol";

contract LockboxSuperchainERC20 is SuperchainERC20 {
    string private _name;
    string private _symbol;
    uint8 private immutable _decimals;
    address immutable _originalTokenAddress;
    uint256 immutable _originalChainId;

    constructor(
        string memory name_, 
        string memory symbol_, 
        uint8 decimals_,
        address originalTokenAddress_,
        uint256 originalChainId_) {
        require(originalTokenAddress_ != address(0), "Invalid token address");
        require(originalChainId_ != 0, "Invalid chain ID");
        _name = name_;
        _symbol = symbol_;
        _decimals = decimals_;
        _originalTokenAddress = originalTokenAddress_;
        _originalChainId = originalChainId_;
    }

    function name() public view virtual override returns (string memory) {
        return _name;
    }

    function symbol() public view virtual override returns (string memory) {
        return _symbol;
    }

    function decimals() public view override returns (uint8) {
        return _decimals;
    }

    function originalTokenAddress() public view returns (address) {
        return _originalTokenAddress;
    }

    function originalChainId() public view returns (uint256) {
        return _originalChainId;
    }

    function lockAndMint(uint256 amount_) external {
        IERC20 originalToken = IERC20(_originalTokenAddress);

        require(block.chainid == _originalChainId, "Wrong chain");
        bool success = originalToken.transferFrom(msg.sender, address(this), amount_);

        // Not necessarily if the ERC-20 contract reverts rather than reverting.
        // However, the standard allows the ERC-20 contract to return false instead.
        require(success, "No tokens to lock, no mint either");
        _mint(msg.sender, amount_);
    }

    function redeemAndBurn(uint256 amount_) external {
        IERC20 originalToken = IERC20(_originalTokenAddress);

        require(block.chainid == _originalChainId, "Wrong chain");
        _burn(msg.sender, amount_);

        bool success = originalToken.transfer(msg.sender, amount_);
        require(success, "Transfer failed, this should not happen");  
    }
}

