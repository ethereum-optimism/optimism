// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";
import { ISuperchainERC20Extensions } from "./ISuperchainERC20.sol";

/// @title IOptimismSuperchainERC20Extension
/// @notice This interface is available on the OptimismSuperchainERC20 contract.
///         We declare it as a separate interface so that it can be used in
///         custom implementations of SuperchainERC20.
interface IOptimismSuperchainERC20Extension is ISuperchainERC20Extensions {
    /// @notice Emitted whenever tokens are minted for an account.
    /// @param account Address of the account tokens are being minted for.
    /// @param amount  Amount of tokens minted.
    event Mint(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are burned from an account.
    /// @param account Address of the account tokens are being burned from.
    /// @param amount  Amount of tokens burned.
    event Burn(address indexed account, uint256 amount);

    /// @notice Allows the L2StandardBridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external;

    /// @notice Allows the L2StandardBridge to burn tokens.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _amount) external;

    /// @notice Returns the address of the corresponding version of this token on the remote chain.
    function remoteToken() external view returns (address);
}

/// @title IOptimismSuperchainERC20
/// @notice Combines the ERC20 interface with the OptimismSuperchainERC20Extension interface.
interface IOptimismSuperchainERC20 is IERC20, IOptimismSuperchainERC20Extension { }
