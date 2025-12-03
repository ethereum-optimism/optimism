// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IGasToken
/// @notice Implemented by contracts that are aware of the custom gas token used
///         by the L2 network.
interface IGasToken {
    /// @notice Getter for the ERC20 token address that is used to pay for gas and its decimals.
    function gasPayingToken() external view returns (address, uint8);
    /// @notice Returns the gas token name.
    function gasPayingTokenName() external view returns (string memory);
    /// @notice Returns the gas token symbol.
    function gasPayingTokenSymbol() external view returns (string memory);
    /// @notice Returns true if the network uses a custom gas token.
    function isCustomGasToken() external view returns (bool);
}