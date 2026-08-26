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

/// @custom:proxied true
/// @custom:predeploy 0x4200000000000000000000000000000000000024
/// @title SuperchainETHBridge
/// @notice SuperchainETHBridge enables ETH transfers between chains within an interop cluster.
contract SuperchainETHBridge is ISemver {
    /// @notice Thrown when attempting to relay a message and the cross domain message sender is not
    /// SuperchainETHBridge.
    error InvalidCrossDomainSender();

    /// @notice Thrown when relaying ETH from a source chain that has not been sent at least that
    /// much net ETH from this chain.
    error InsufficientNetFlow();

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
    /// @custom:semver 1.1.0
    function version() public pure virtual returns (string memory) {
        return "1.1.0";
    }

    /// @notice Net ETH this chain has sent to each counterpart chain and that has not been relayed
    ///         back yet. This is the per-counterparty solvency cap: a chain can never mint more ETH
    ///         here than was voluntarily sent to it, so the worst case loss from a counterpart chain
    ///         lying about its exported messages is exactly the ETH sent to that chain. A chain
    ///         nobody has sent ETH to has a cap of zero and can extract nothing.
    mapping(uint256 => uint256) public netSent;

    /// @notice Sends ETH to some target address on another chain.
    /// @param _to       Address to send ETH to.
    /// @param _chainId  Chain ID of the destination chain.
    /// @return msgHash_ Hash of the message sent.
    function sendETH(address _to, uint256 _chainId) public payable virtual returns (bytes32 msgHash_) {
        if (_to == address(0)) revert ZeroAddress();

        // Credit the destination chain: it may now relay back at most this much more ETH.
        netSent[_chainId] += msg.value;

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
    function relayETH(address _from, address _to, uint256 _amount) public virtual {
        if (msg.sender != Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER) revert Unauthorized();

        (address crossDomainMessageSender, uint256 source) =
            IL2ToL2CrossDomainMessenger(Predeploys.L2_TO_L2_CROSS_DOMAIN_MESSENGER).crossDomainMessageContext();

        if (crossDomainMessageSender != address(this)) revert InvalidCrossDomainSender();

        // Debit the source chain's cap. Bounds the ETH any counterpart chain can ever mint here by
        // the ETH this chain voluntarily sent to it.
        if (netSent[source] < _amount) revert InsufficientNetFlow();
        netSent[source] -= _amount;

        // NOTE: 'mint' will soon change to 'withdraw'.
        IETHLiquidity(Predeploys.ETH_LIQUIDITY).mint(_amount);

        // This is a forced ETH send to the recipient, the recipient should NOT expect to be called.
        new SafeSend{ value: _amount }(payable(_to));

        emit RelayETH(_from, _to, _amount, source);
    }
}
