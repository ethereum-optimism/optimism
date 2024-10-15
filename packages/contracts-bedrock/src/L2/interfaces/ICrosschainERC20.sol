// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title ICrosschainERC20
/// @notice Defines the interface for crosschain ERC20 transfers.
interface ICrosschainERC20 {
    /// @notice Emitted when a crosschain transfer mints tokens.
    /// @param to       Address of the account tokens are being minted for.
    /// @param amount   Amount of tokens minted.
    event CrosschainMinted(address indexed to, uint256 amount);

    /// @notice Emitted when a crosschain transfer burns tokens.
    /// @param from     Address of the account tokens are being burned from.
    /// @param amount   Amount of tokens burned.
    event CrosschainBurnt(address indexed from, uint256 amount);

    /// @notice Mint tokens through a crosschain transfer.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function crosschainMint(address _to, uint256 _amount) external;

    /// @notice Burn tokens through a crosschain transfer.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function crosschainBurn(address _from, uint256 _amount) external;
}
