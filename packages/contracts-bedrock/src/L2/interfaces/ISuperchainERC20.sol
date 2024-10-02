// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { IERC20Solady } from "src/vendor/interfaces/IERC20Solady.sol";

/// @title ISuperchainERC20Errors
/// @notice Interface containing the errors added in the SuperchainERC20 implementation.
interface ISuperchainERC20Errors {
    /// @notice Thrown when attempting to mint or burn tokens and the function caller is not the SuperchainERC20Bridge.
    error OnlySuperchainERC20Bridge();
}

/// @title ISuperchainERC20Extension
/// @notice This interface is available on the SuperchainERC20 contract.
interface ISuperchainERC20Extension is ISuperchainERC20Errors {
    /// @notice Emitted whenever tokens are minted for by the SuperchainERC20Bridge.
    /// @param account Address of the account tokens are being minted for.
    /// @param amount  Amount of tokens minted.
    event SuperchainMinted(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are burned by the SuperchainERC20Bridge.
    /// @param account Address of the account tokens are being burned from.
    /// @param amount  Amount of tokens burned.
    event SuperchainBurnt(address indexed account, uint256 amount);

    /// @notice Allows the SuperchainERC20Bridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function __superchainMint(address _to, uint256 _amount) external;

    /// @notice Allows the SuperchainERC20Bridge to burn tokens.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function __superchainBurn(address _from, uint256 _amount) external;
}

/// @title ISuperchainERC20
/// @notice Combines Solady's ERC20 interface with the SuperchainERC20Extension interface.
interface ISuperchainERC20 is IERC20Solady, ISuperchainERC20Extension { }
