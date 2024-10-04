// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

// Interfaces
import { ISuperchainERC20 } from "src/L2/interfaces/ISuperchainERC20.sol";

/// @title IOptimismSuperchainERC20
/// @notice This interface is available on the OptimismSuperchainERC20 contract.
interface IOptimismSuperchainERC20 is ISuperchainERC20 {
    /// @notice Thrown when attempting to perform an operation and the account is the zero address.
    error ZeroAddress();

    /// @notice Thrown when attempting to mint or burn tokens and the function caller is not the L2StandardBridge
    error OnlyL2StandardBridge();

    /// @notice Emitted whenever tokens are minted for an account.
    /// @param to Address of the account tokens are being minted for.
    /// @param amount  Amount of tokens minted.
    event Mint(address indexed to, uint256 amount);

    /// @notice Emitted whenever tokens are burned from an account.
    /// @param from Address of the account tokens are being burned from.
    /// @param amount  Amount of tokens burned.
    event Burn(address indexed from, uint256 amount);

    /// @notice Allows the L2StandardBridge and SuperchainERC20Bridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external;

    /// @notice Allows the L2StandardBridge and SuperchainERC20Bridge to burn tokens.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _amount) external;

    /// @notice Returns the address of the corresponding version of this token on the remote chain.
    function remoteToken() external view returns (address);
}
