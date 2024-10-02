// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { ISemver } from "src/universal/interfaces/ISemver.sol";

/// @title ISuperchainERC20Bridge
/// @notice Interface for the SuperchainERC20Bridge contract.
interface ISuperchainERC20Bridge is ISemver {
    /// @notice Thrown when attempting to perform an operation and the account is the zero address.
    error ZeroAddress();

    /// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not
    /// L2ToL2CrossDomainMessenger.
    error CallerNotL2ToL2CrossDomainMessenger();

    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not the
    /// SuperchainERC20Bridge.
    error InvalidCrossDomainSender();

    /// @notice Emitted when tokens are sent from one chain to another.
    /// @param token         Address of the token sent.
    /// @param from          Address of the sender.
    /// @param to            Address of the recipient.
    /// @param amount        Number of tokens sent.
    /// @param destination   Chain ID of the destination chain.
    event SendERC20(
        address indexed token, address indexed from, address indexed to, uint256 amount, uint256 destination
    );

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param token         Address of the token relayed.
    /// @param from          Address of the msg.sender of sendERC20 on the source chain.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of tokens relayed.
    /// @param source        Chain ID of the source chain.
    event RelayERC20(address indexed token, address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Sends tokens to some target address on another chain.
    /// @param _token   Token to send.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _token, address _to, uint256 _amount, uint256 _chainId) external;

    /// @notice Relays tokens received from another chain.
    /// @param _token   Token to relay.
    /// @param _from    Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to      Address to relay tokens to.
    /// @param _amount  Amount of tokens to relay.
    function relayERC20(address _token, address _from, address _to, uint256 _amount) external;
}
