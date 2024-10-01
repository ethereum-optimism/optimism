// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

// Libraries
import { Predeploys } from "src/libraries/Predeploys.sol";

// Interfaces
import { ISuperchainERC20Bridge } from "src/L2/interfaces/ISuperchainERC20Bridge.sol";
import { ISuperchainERC20 } from "src/L2/interfaces/ISuperchainERC20.sol";
import { IL2ToL2CrossDomainMessenger } from "src/L2/interfaces/IL2ToL2CrossDomainMessenger.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000028
/// @title SuperchainERC20Bridge
/// @notice The SuperchainERC20Bridge allows for the bridging of ERC20 tokens to make them fungible across the
///         Superchain. It builds on top of the L2ToL2CrossDomainMessenger for both replay protection and domain
///         binding.
contract SuperchainERC20Bridge is ISuperchainERC20Bridge {
    /// @notice Address of the L2ToL2CrossDomainMessenger Predeploy.
    address internal constant MESSENGER = Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER;

    /// @notice Semantic version.
    /// @custom:semver 1.0.0-beta.1
    string public constant version = "1.0.0-beta.1";

    /// @notice Sends tokens to some target address on another chain.
    /// @param _token   Token to send.
    /// @param _to      Address to send tokens to.
    /// @param _amount  Amount of tokens to send.
    /// @param _chainId Chain ID of the destination chain.
    function sendERC20(address _token, address _to, uint256 _amount, uint256 _chainId) external {
        ISuperchainERC20(_token).__superchainBurn(msg.sender, _amount);

        bytes memory message = abi.encodeCall(this.relayERC20, (_token, msg.sender, _to, _amount));
        IL2ToL2CrossDomainMessenger(MESSENGER).sendMessage(_chainId, address(this), message);

        emit SendERC20(_token, msg.sender, _to, _amount, _chainId);
    }

    /// @notice Relays tokens received from another chain.
    /// @param _token  Token to relay.
    /// @param _from   Address of the msg.sender of sendERC20 on the source chain.
    /// @param _to     Address to relay tokens to.
    /// @param _amount Amount of tokens to relay.
    function relayERC20(address _token, address _from, address _to, uint256 _amount) external {
        if (msg.sender != MESSENGER) revert CallerNotL2ToL2CrossDomainMessenger();

        if (IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSender() != address(this)) {
            revert InvalidCrossDomainSender();
        }

        uint256 source = IL2ToL2CrossDomainMessenger(MESSENGER).crossDomainMessageSource();

        ISuperchainERC20(_token).__superchainMint(_to, _amount);

        emit RelayERC20(_token, _from, _to, _amount, source);
    }
}
