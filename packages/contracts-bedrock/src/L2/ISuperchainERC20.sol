// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IERC20 } from "@openzeppelin/contracts/token/ERC20/IERC20.sol";

/// @title ISuperchainERC20Extensions
/// @notice Interface for the extensions to the ERC20 standard that are used by SuperchainERC20.
///         Exists in case developers are already importing the ERC20 interface separately and
///         importing the full SuperchainERC20 interface would cause conflicting imports.
interface ISuperchainERC20Extensions {
    /// @notice Emitted when tokens are sent from one chain to another.
    /// @param from         Address of the sender.
    /// @param to           Address of the recipient.
    /// @param amount       Number of tokens sent.
    /// @param destination  Chain ID of the destination chain.
    event SendERC20(address indexed from, address indexed to, uint256 amount, uint256 destination);

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param from     Address of the msg.sender of sendERC20 on the source chain.
    /// @param to       Address of the recipient.
    /// @param amount   Amount of tokens relayed.
    /// @param source   Chain ID of the source chain.
    event RelayERC20(address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Sends tokens to some target address on another chain.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _to, uint256 _amount, uint256 _chainId) external;

    /// @notice Relays tokens received from another chain.
    /// @param _from    Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to      Address to relay tokens to.
    /// @param _amount  Amount of tokens to relay.
    function relayERC20(address _from, address _to, uint256 _amount) external;
}

/// @title ISuperchainERC20
/// @notice Combines the ERC20 interface with the SuperchainERC20Extensions interface.
interface ISuperchainERC20 is IERC20, ISuperchainERC20Extensions { }
