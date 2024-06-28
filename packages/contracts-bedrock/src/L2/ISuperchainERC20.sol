// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title ISuperchainERC20Extensions
/// @notice Interface for the extensions to the ERC20 standard that are used by SuperchainERC20.
///         Exists in case developers are already importing the ERC20 interface separately and
///         importing the full SuperchainERC20 interface would cause conflicting imports.
interface ISuperchainERC20Extensions {
    /// @notice Emitted when tokens are sent from one chain to another.
    /// @param _from    Address of the sender.
    /// @param _to      Address of the recipient.
    /// @param _amount  Number of tokens sent.
    /// @param _chainId Chain ID of the recipient.
    event SendERC20(address indexed _from, address indexed _to, uint256 _amount, uint256 _chainId);

    /// @notice Emitted when token sends are relayed to this chain.
    /// @param _to     Address of the recipient.
    /// @param _amount Number of tokens sent.
    event RelayERC20(address indexed _to, uint256 _amount);

    /// @notice Sends tokens to another chain.
    /// @param _to      Address of the recipient.
    /// @param _amount  Number of tokens to send.
    /// @param _chainId Chain ID of the recipient.
    function sendERC20(address _to, uint256 _amount, uint256 _chainId) external;

    /// @notice Relays a send of tokens to this chain.
    /// @param _to     Address of the recipient.
    /// @param _amount Number of tokens sent.
    function relayERC20(address _to, uint256 _amount) external;
}

/// @title ISuperchainERC20
/// @notice Combines the ERC20 interface with the SuperchainERC20Extensions interface.
interface ISuperchainERC20 is IERC20, ISuperchainERC20Extensions { }
