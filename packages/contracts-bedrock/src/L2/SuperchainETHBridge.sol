// SPDX-License-Identifier: MIT
pragma solidity 0.8.15;

// Libraries
import { Unauthorized, ZeroAddress } from "src/libraries/errors/CommonErrors.sol";
import { Predeploys } from "src/libraries/Predeploys.sol";
import { SafeSend } from "src/universal/SafeSend.sol";

// Interfaces
import { ISemver } from "interfaces/universal/ISemver.sol";
import { IL2ToL2CrossDomainMessenger } from "interfaces/L2/IL2ToL2CrossDomainMessenger.sol";
import { IETHLiquidity } from "interfaces/L2/IETHLiquidity.sol";
import { IProxyAdmin } from "interfaces/universal/IProxyAdmin.sol";

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000024
/// @title SuperchainETHBridge
/// @notice SuperchainETHBridge enables ETH transfers between chains within an interop cluster.
contract SuperchainETHBridge is ISemver {
    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not
    /// SuperchainETHBridge.
    error InvalidCrossDomainSender();

    /// @notice The destination is not approved for native ETH sends.
    error SuperchainETHBridge_SendChainNotAllowed();

    /// @notice The authenticated source is not approved for native ETH relays.
    error SuperchainETHBridge_RelayChainNotAllowed();

    /// @notice Native ETH destinations approved by the L2 ProxyAdmin owner.
    mapping(uint256 => bool) public allowedSendChain;

    /// @notice Native ETH sources approved by the L2 ProxyAdmin owner.
    mapping(uint256 => bool) public allowedRelayChain;

    /// @notice Emitted when the native ETH permissions for a chain change.
    event ChainPermissionsUpdated(uint256 indexed chainId, bool allowSend, bool allowRelay);

    /// @notice Emitted when ETH is sent from one chain to another.
    /// @param from          Address of the sender.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of ETH sent.
    /// @param destination   Chain ID of the destination chain.
    event SendETH(address indexed from, address indexed to, uint256 amount, uint256 destination);

    /// @notice Emitted whenever ETH is successfully relayed on this chain.
    /// @param from          Address of the msg.sender of sendETH on the source chain.
    /// @param to            Address of the recipient.
    /// @param amount        Amount of ETH relayed.
    /// @param source        Chain ID of the source chain.
    event RelayETH(address indexed from, address indexed to, uint256 amount, uint256 source);

    /// @notice Semantic version.
    /// @custom:semver 2.0.0
    string public constant version = "2.0.0";

    /// @notice Sets native ETH permissions. Governance must verify shared backing before enabling
    ///         a route. Stop new sends and drain pending transfers before revoking relay permission.
    ///
    /// @param _chainId Chain whose permissions are being configured.
    /// @param _allowSend Whether this chain may send native ETH to that chain.
    /// @param _allowRelay Whether this chain may release native ETH for messages from that chain.
    function setChainPermissions(uint256 _chainId, bool _allowSend, bool _allowRelay) external {
        if (msg.sender != IProxyAdmin(Predeploys.PROXY_ADMIN).owner()) revert Unauthorized();
        allowedSendChain[_chainId] = _allowSend;
        allowedRelayChain[_chainId] = _allowRelay;
        emit ChainPermissionsUpdated(_chainId, _allowSend, _allowRelay);
    }

    /// @notice Sends ETH to some target address on another chain.
    /// @param _to       Address to send ETH to.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendETH(address _to, uint256 _chainId) external payable returns (bytes32 msgHash_) {
        if (_to == address(0)) revert ZeroAddress();
        if (!allowedSendChain[_chainId]) revert SuperchainETHBridge_SendChainNotAllowed();

        // NOTE: 'burn' will soon change to 'deposit'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).burn{ value: msg.value }();

        msgHash_ = IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).sendMessage({
            _destination: _chainId,
            _target: address(this),
            _message: abi.encodeCall(this.relayETH, (msg.sender, _to, msg.value))
        });

        emit SendETH(msg.sender, _to, msg.value, _chainId);
    }

    /// @notice Relays ETH received from another chain.
    /// @param _from       Address of the msg.sender of sendETH on the source chain.
    /// @param _to         Address to relay ETH to.
    /// @param _amount     Amount of ETH to relay.
    function relayETH(address _from, address _to, uint256 _amount) external {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert InvalidCrossDomainSender();
        if (!allowedRelayChain[source]) revert SuperchainETHBridge_RelayChainNotAllowed();

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(_amount);

        // This is a forced ETH send to the recipient, the recipient should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_to));

        emit RelayETH(_from, _to, _amount, source);
    }
}
