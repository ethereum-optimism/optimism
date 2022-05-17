// SPDX-License-Identifier: MIT
pragma solidity ^0.8.9;

import { Owned } from "@rari-capital/solmate/src/auth/Owned.sol";
import { ERC20 } from "@rari-capital/solmate/src/tokens/ERC20.sol";
import { ERC721 } from "@rari-capital/solmate/src/tokens/ERC721.sol";

/**
 * @title RetroReceiver
 * @notice RetroReceiver is a minimal contract for receiving funds, meant to be deployed at the
 * same address on every chain that supports EIP-2470.
 */
contract RetroReceiver is Owned {
    /**
     * Emitted when ETH is received by this address.
     */
    event ReceivedETH(address indexed from, uint256 amount);

    /**
     * Emitted when ETH is withdrawn from this address.
     */
    event WithdrewETH(address indexed withdrawer, address indexed recipient, uint256 amount);

    /**
     * Emitted when ERC20 tokens are withdrawn from this address.
     */
    event WithdrewERC20(
        address indexed withdrawer,
        address indexed recipient,
        address indexed asset,
        uint256 amount
    );

    /**
     * Emitted when ERC721 tokens are withdrawn from this address.
     */
    event WithdrewERC721(
        address indexed withdrawer,
        address indexed recipient,
        address indexed asset,
        uint256 id
    );

    /**
     * @param _owner Address to initially own the contract.
     */
    constructor(address _owner) Owned(_owner) {}

    /**
     * Make sure we can receive ETH.
     */
    receive() external payable {
        emit ReceivedETH(msg.sender, msg.value);
    }

    /**
     * Withdraws full ETH balance to the recipient.
     *
     * @param _to Address to receive the ETH balance.
     */
    function withdrawETH(address payable _to) public onlyOwner {
        withdrawETH(_to, address(this).balance);
    }

    /**
     * Withdraws partial ETH balance to the recipient.
     *
     * @param _to Address to receive the ETH balance.
     * @param _amount Amount of ETH to withdraw.
     */
    function withdrawETH(address payable _to, uint256 _amount) public onlyOwner {
        _to.transfer(_amount);
        emit WithdrewETH(msg.sender, _to, _amount);
    }

    /**
     * Withdraws full ERC20 balance to the recipient.
     *
     * @param _asset ERC20 token to withdraw.
     * @param _to Address to receive the ERC20 balance.
     */
    function withdrawERC20(ERC20 _asset, address _to) public onlyOwner {
        withdrawERC20(_asset, _to, _asset.balanceOf(address(this)));
    }

    /**
     * Withdraws partial ERC20 balance to the recipient.
     *
     * @param _asset ERC20 token to withdraw.
     * @param _to Address to receive the ERC20 balance.
     * @param _amount Amount of ERC20 to withdraw.
     */
    function withdrawERC20(
        ERC20 _asset,
        address _to,
        uint256 _amount
    ) public onlyOwner {
        _asset.transfer(_to, _amount);
        emit WithdrewERC20(msg.sender, _to, address(_asset), _amount);
    }

    /**
     * Withdraws ERC721 token to the recipient.
     *
     * @param _asset ERC721 token to withdraw.
     * @param _to Address to receive the ERC721 token.
     * @param _id Token ID of the ERC721 token to withdraw.
     */
    function withdrawERC721(
        ERC721 _asset,
        address _to,
        uint256 _id
    ) public onlyOwner {
        _asset.transferFrom(address(this), _to, _id);
        emit WithdrewERC721(msg.sender, _to, address(_asset), _id);
    }
}
