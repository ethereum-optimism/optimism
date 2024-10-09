// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

import { ISuperchainERC20Extensions, ISuperchainERC20Errors } from "src/L2/interfaces/ISuperchainERC20.sol";
import { ERC20 } from "@solady/tokens/ERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";

/// @title SuperchainERC20
/// @notice SuperchainERC20 is a standard extension of the base ERC20 token contract that unifies ERC20 token
///         bridging to make it fungible across the Superchain. It builds on top of the L2ToL2CrossDomainMessenger for
///         both replay protection and domain binding.
abstract contract SuperchainERC20 is ISuperchainERC20Extensions, ISuperchainERC20Errors, ERC20 {
    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Sends tokens to some target address on another chain.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _to, uint256 _amount, uint256 _chainId) external virtual {
        if (_to == address(0)) revert ZeroAddress();

        _burn(msg.sender, _amount);

        bytes memory _message = abi.encodeCall(this.relayERC20, (msg.sender, _to, _amount));
        IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_chainId, address(this), _message);

        emit SendERC20(msg.sender, _to, _amount, _chainId);
    }

    /// @notice Relays tokens received from another chain.
    /// @param _from   Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to     Address to relay tokens to.
    /// @param _amount Amount of tokens to relay.
    function relayERC20(address _from, address _to, uint256 _amount) external virtual {
        if (msg.sender != MESSENGER) revert CallerNotL2ToL2CrossDomainMessenger();

        if (IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSender() != address(this)) {
            revert InvalidCrossDomainSender();
        }

        uint256 source = IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSource();

        _mint(_to, _amount);

        emit RelayERC20(_from, _to, _amount, source);
    }
}
