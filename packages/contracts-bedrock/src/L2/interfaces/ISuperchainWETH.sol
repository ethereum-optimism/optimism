// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

import { IWETH } from "src/universal/interfaces/IWETH.sol";

interface ISuperchainWETH {
    /// @notice Thrown when attempting a deposit or withdrawal and the chain uses a custom gas token.
    error NotCustomGasToken();

    /// @notice Thrown when attempting to relay a message and the function caller (msg.sender) is not
    /// L2ToL2CrossDomainMessenger.
    error CallerNotL2ToL2CrossDomainMessenger();

    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not `address(this)`
    error InvalidCrossDomainSender();

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param from     Address of the msg.sender of sendERC20 on the source chain.
    /// @param to       Address of the recipient.
    /// @param amount   Amount of tokens relayed.
    /// @param source   Chain ID of the source chain.
    event RelayERC20(address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Emitted when tokens are sent from one chain to another.
    /// @param from         Address of the sender.
    /// @param to           Address of the recipient.
    /// @param amount       Number of tokens sent.
    /// @param destination  Chain ID of the destination chain.
    event SendERC20(address indexed from, address indexed to, uint256 amount, uint256 destination);

    /// @notice Sends tokens to some target address on another chain.
    /// @param _dst      Address to send tokens to.
    /// @param _wad      Amount of tokens to send.
    /// @param _chainId  Chain ID of the destination chain.
    function sendERC20(address _dst, uint256 _wad, uint256 _chainId) external;

    /// @notice Relays tokens received from another chain.
    /// @param _from    Address of the msg.sender of sendERC20 on the source chain.
    /// @param _dst     Address to relay tokens to.
    /// @param _wad     Amount of tokens to relay.
    function relayERC20(address _from, address _dst, uint256 _wad) external;
}

interface ISuperchainWETHERC20 is IWETH, ISuperchainWETH { }
