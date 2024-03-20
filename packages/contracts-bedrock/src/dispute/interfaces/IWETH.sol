// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IWETH
/// @notice Interface for WETH9.
interface IWETH {
    /// @notice Emitted when an approval is made.
    /// @param src The address that approved the transfer.
    /// @param guy The address that was approved to transfer.
    /// @param wad The amount that was approved to transfer.
    event Approval(address indexed src, address indexed guy, uint256 wad);

    /// @notice Emitted when a transfer is made.
    /// @param src The address that transferred the WETH.
    /// @param dst The address that received the WETH.
    /// @param wad The amount of WETH that was transferred.
    event Transfer(address indexed src, address indexed dst, uint256 wad);

    /// @notice Emitted when a deposit is made.
    /// @param dst The address that deposited the WETH.
    /// @param wad The amount of WETH that was deposited.
    event Deposit(address indexed dst, uint256 wad);

    /// @notice Emitted when a withdrawal is made.
    /// @param src The address that withdrew the WETH.
    /// @param wad The amount of WETH that was withdrawn.
    event Withdrawal(address indexed src, uint256 wad);

    /// @notice Returns the name of the token.
    /// @return The name of the token.
    function name() external pure returns (string memory);

    /// @notice Returns the symbol of the token.
    /// @return The symbol of the token.
    function symbol() external pure returns (string memory);

    /// @notice Returns the number of decimals the token uses.
    /// @return The number of decimals the token uses.
    function decimals() external pure returns (uint8);

    /// @notice Returns the balance of the given address.
    /// @param owner The address to query the balance of.
    /// @return The balance of the given address.
    function balanceOf(address owner) external view returns (uint256);

    /// @notice Returns the amount of WETH that the spender can transfer on behalf of the owner.
    /// @param owner The address that owns the WETH.
    /// @param spender The address that is approved to transfer the WETH.
    /// @return The amount of WETH that the spender can transfer on behalf of the owner.
    function allowance(address owner, address spender) external view returns (uint256);

    /// @notice Allows WETH to be deposited by sending ether to the contract.
    function deposit() external payable;

    /// @notice Withdraws an amount of ETH.
    /// @param wad The amount of ETH to withdraw.
    function withdraw(uint256 wad) external;

    /// @notice Returns the total supply of WETH.
    /// @return The total supply of WETH.
    function totalSupply() external view returns (uint256);

    /// @notice Approves the given address to transfer the WETH on behalf of the caller.
    /// @param guy The address that is approved to transfer the WETH.
    /// @param wad The amount that is approved to transfer.
    /// @return True if the approval was successful.
    function approve(address guy, uint256 wad) external returns (bool);

    /// @notice Transfers the given amount of WETH to the given address.
    /// @param dst The address to transfer the WETH to.
    /// @param wad The amount of WETH to transfer.
    /// @return True if the transfer was successful.
    function transfer(address dst, uint256 wad) external returns (bool);

    /// @notice Transfers the given amount of WETH from the given address to the given address.
    /// @param src The address to transfer the WETH from.
    /// @param dst The address to transfer the WETH to.
    /// @param wad The amount of WETH to transfer.
    /// @return True if the transfer was successful.
    function transferFrom(address src, address dst, uint256 wad) external returns (bool);
}
