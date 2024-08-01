// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title IOptimismSuperchainERC20
/// @notice This interface is available on the OptimismSuperchainERC20 contract.
///         We declare it as a separate interface so that it can be used in
///         custom implementations of SuperchainERC20.
interface IOptimismSuperchainERC20 {
    /// @notice Emitted whenever tokens are minted for an account.
    /// @param account Address of the account tokens are being minted for.
    /// @param amount  Amount of tokens minted.
    event Mint(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are burned from an account.
    /// @param account Address of the account tokens are being burned from.
    /// @param amount  Amount of tokens burned.
    event Burn(address indexed account, uint256 amount);

    /// @notice Emitted whenever tokens are sent to another chain.
    /// @param from         Address of the sender.
    /// @param to           Address of the recipient.
    /// @param amount       Amount of tokens sent.
    /// @param destination  Chain ID of the destination chain.
    event SendERC20(address indexed from, address indexed to, uint256 amount, uint256 destination);

    /// @notice Emitted whenever tokens are successfully relayed on this chain.
    /// @param from     Address of the msg.sender of sendERC20 on the source chain.
    /// @param to       Address of the recipient.
    /// @param amount   Amount of tokens relayed.
    /// @param source   Chain ID of the source chain.
    event RelayedERC20(address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Allows the L2StandardBridge to mint tokens.
    /// @param _to     Address to mint tokens to.
    /// @param _amount Amount of tokens to mint.
    function mint(address _to, uint256 _amount) external;

    /// @notice Allows the L2StandardBridge to burn tokens.
    /// @param _from   Address to burn tokens from.
    /// @param _amount Amount of tokens to burn.
    function burn(address _from, uint256 _amount) external;

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

    /// @notice Returns the address of the corresponding version of this token on the remote chain.
    function remoteToken() external view returns (address);
}
